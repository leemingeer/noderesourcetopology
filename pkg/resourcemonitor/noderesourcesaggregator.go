package resourcemonitor

import (
	"context"
	"fmt"
	"github.com/jaypipes/ghw"
	topologyv1alpha1 "github.com/leemingeer/noderesourcetopology/pkg/apis/topology/v1alpha1"
	"github.com/leemingeer/noderesourcetopology/pkg/utils"
	"github.com/leemingeer/noderesourcetopology/pkg/utils/hostpath"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"
	podresourcesapi "k8s.io/kubelet/pkg/apis/podresources/v1"
	"strconv"
	"strings"
)

func NewResourcesAggregator(podResourceClient podresourcesapi.PodResourcesListerClient, excludeList ExcludeResourceList) (ResourcesAggregator, error) {
	var err error

	topo, err := ghw.Topology(ghw.WithPathOverrides(ghw.PathOverrides{
		"/sys": string(hostpath.SysfsDir),
	}))
	if err != nil {
		return nil, err
	}

	memoryResourcesCapacityPerNUMA, err := getMemoryResourcesCapacity()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultPodResourcesTimeout)
	defer cancel()

	// Pod Resource API client
	resp, err := podResourceClient.GetAllocatableResources(ctx, &podresourcesapi.AllocatableResourcesRequest{})
	if err != nil {
		if strings.Contains(err.Error(), "API GetAllocatableResources disabled") {
			klog.ErrorS(err, "Kubelet's pod resources 'GetAllocatableResources' functionality is disabled. "+
				"Ensure feature flag 'KubeletPodResourcesGetAllocatable' is set to true. "+
				"You can find more about the feature gates from the following URL - "+
				"https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/")
		}

		return nil, fmt.Errorf("failed to get allocatable resources (ensure that KubeletPodResourcesGetAllocatable feature gate is enabled): %w", err)
	}

	return NewResourcesAggregatorFromData(topo, resp, memoryResourcesCapacityPerNUMA, excludeList), nil
}

func MakeLogicalCoreIDToNodeIDMap(topo *ghw.TopologyInfo) map[int]int {
	core2node := make(map[int]int)
	for _, node := range topo.Nodes {
		for _, core := range node.Cores {
			for _, procID := range core.LogicalProcessors {
				core2node[procID] = node.ID
			}
		}
	}
	return core2node
}

// getContainerDevicesFromAllocatableResources normalize all compute resources to ContainerDevices.
// This is helpful because cpuIDs are not represented as ContainerDevices, but with a different format;
// Having a consistent representation of all the resources as ContainerDevices makes it simpler for
func getContainerDevicesFromAllocatableResources(availRes *podresourcesapi.AllocatableResourcesResponse, topo *ghw.TopologyInfo) []*podresourcesapi.ContainerDevices {
	var contDevs []*podresourcesapi.ContainerDevices
	contDevs = append(contDevs, availRes.GetDevices()...)

	cpuIDToNodeIDMap := MakeLogicalCoreIDToNodeIDMap(topo)

	cpusPerNuma := make(map[int][]string)
	for _, cpuID := range availRes.GetCpuIds() {
		nodeID, ok := cpuIDToNodeIDMap[int(cpuID)]
		if !ok {
			klog.InfoS("failed to find the NUMA node for CPU", "cpuID", cpuID)
			continue
		}

		cpuIDList := cpusPerNuma[nodeID]
		cpuIDList = append(cpuIDList, fmt.Sprintf("%d", cpuID))
		cpusPerNuma[nodeID] = cpuIDList
	}

	for nodeID, cpuList := range cpusPerNuma {
		contDevs = append(contDevs, &podresourcesapi.ContainerDevices{
			ResourceName: string(corev1.ResourceCPU),
			DeviceIds:    cpuList,
			Topology: &podresourcesapi.TopologyInfo{
				Nodes: []*podresourcesapi.NUMANode{
					{ID: int64(nodeID)},
				},
			},
		})
	}

	return contDevs
}

type nodeResources struct {
	perNUMAAllocatable map[int]map[corev1.ResourceName]int64
	// mapping: resourceName -> resourceID -> nodeID
	resourceID2NUMAID              map[string]map[string]int
	topo                           *ghw.TopologyInfo
	reservedCPUIDPerNUMA           map[int][]string
	memoryResourcesCapacityPerNUMA utils.NumaMemoryResources
	excludeList                    ExcludeResourceList
}

type resourceData struct {
	available   int64
	allocatable int64
	capacity    int64
}

// updateMemoryAvailable computes the actual amount of the available memory.
// This function assumes the available resources are initialized to be equal to the capacity.
func (noderesourceData *nodeResources) updateMemoryAvailable(numaData map[int]map[corev1.ResourceName]*resourceData, ri ResourceInfo) {
	if len(ri.NumaNodeIds) == 0 {
		klog.InfoS("no NUMA nodes information is available", "resourceName", ri.Name)
		return
	}

	if len(ri.Data) != 1 {
		klog.InfoS("no size information is available", "resourceName", ri.Name)
		return
	}

	requestedSize, err := strconv.ParseInt(ri.Data[0], 10, 64)
	if err != nil {
		klog.ErrorS(err, "failed to parse resource requested size")
		return
	}

	for _, numaNodeID := range ri.NumaNodeIds {
		if requestedSize == 0 {
			return
		}

		if _, ok := numaData[numaNodeID]; !ok {
			klog.InfoS("failed to find NUMA node ID under the node topology", "numaID", numaNodeID)
			continue
		}

		if _, ok := numaData[numaNodeID][ri.Name]; !ok {
			klog.InfoS("failed to find resource under the node topology", "resourceName", ri.Name)
			return
		}

		if numaData[numaNodeID][ri.Name].available == 0 {
			klog.V(4).InfoS("no available memory", "numaID", numaNodeID, "resourceName", ri.Name)
			continue
		}

		// For the container pinned only to one NUMA node the calculation is pretty straight forward, the code will
		// just reduce the specified NUMA node free size
		// For the container pinned to multiple NUMA nodes, the code will reduce the free size of NUMA nodes
		// in ascending order. For example, for a container pinned to NUMA node 0 and NUMA node 1,
		// it will first reduce the memory of the NUMA node 0 to zero, and after the remaining
		// amount of memory from the NUMA node 1.
		// This behavior is tightly coupled with the Kubernetes memory manager logic.
		if requestedSize >= numaData[numaNodeID][ri.Name].available {
			requestedSize -= numaData[numaNodeID][ri.Name].available
			numaData[numaNodeID][ri.Name].available = 0
		} else {
			numaData[numaNodeID][ri.Name].available -= requestedSize
			requestedSize = 0
		}
	}

	if requestedSize > 0 {
		klog.InfoS("requested size was not fully satisfied by NUMA nodes", "resourceName", ri.Name)
	}
}

// updateAvailable computes the actually available resources.
// This function assumes the available resources are initialized to be equal to the allocatable.
func (noderesourceData *nodeResources) updateAvailable(numaData map[int]map[corev1.ResourceName]*resourceData, ri ResourceInfo) {
	for _, resID := range ri.Data {
		resName := string(ri.Name)
		resMap, ok := noderesourceData.resourceID2NUMAID[resName]
		if !ok {
			klog.InfoS("unknown resource", "resourceName", ri.Name)
			continue
		}
		nodeID, ok := resMap[resID]
		if !ok {
			klog.InfoS("unknown resource", "resourceName", resName, "resourceID", resID)
			continue
		}
		if _, ok := numaData[nodeID]; !ok {
			klog.InfoS("unknown NUMA node id", "numaID", nodeID)
			continue
		}

		numaData[nodeID][ri.Name].available--
	}
}

// Aggregate provides the mapping (numa zone name) -> Zone from the given PodResources.
func (noderesourceData *nodeResources) Aggregate(podResData []PodResources) topologyv1alpha1.ZoneList {
	perNuma := make(map[int]map[corev1.ResourceName]*resourceData)
	for nodeID := range noderesourceData.topo.Nodes {
		nodeRes, ok := noderesourceData.perNUMAAllocatable[nodeID]
		if ok {
			perNuma[nodeID] = make(map[corev1.ResourceName]*resourceData)
			for resName, allocatable := range nodeRes {
				if noderesourceData.excludeList.IsExcluded(resName) {
					continue
				}
				switch {
				case resName == "cpu":
					perNuma[nodeID][resName] = &resourceData{
						allocatable: allocatable,
						available:   allocatable,
						capacity:    allocatable + int64(len(noderesourceData.reservedCPUIDPerNUMA[nodeID])),
					}
				case resName == corev1.ResourceMemory, strings.HasPrefix(string(resName), corev1.ResourceHugePagesPrefix):
					var capacity int64
					if _, ok := noderesourceData.memoryResourcesCapacityPerNUMA[nodeID]; !ok {
						capacity = allocatable
					} else if _, ok := noderesourceData.memoryResourcesCapacityPerNUMA[nodeID][resName]; !ok {
						capacity = allocatable
					} else {
						capacity = noderesourceData.memoryResourcesCapacityPerNUMA[nodeID][resName]
					}

					perNuma[nodeID][resName] = &resourceData{
						allocatable: allocatable,
						available:   allocatable,
						capacity:    capacity,
					}
				default:
					perNuma[nodeID][resName] = &resourceData{
						allocatable: allocatable,
						available:   allocatable,
						capacity:    allocatable,
					}
				}
			}
			// NUMA node doesn't have any allocatable resources, but yet it exists in the topology
			// thus all its CPUs are reserved
		} else {
			perNuma[nodeID] = make(map[corev1.ResourceName]*resourceData)
			perNuma[nodeID]["cpu"] = &resourceData{
				allocatable: int64(0),
				available:   int64(0),
				capacity:    int64(len(noderesourceData.reservedCPUIDPerNUMA[nodeID])),
			}
		}
	}

	for _, podRes := range podResData {
		for _, contRes := range podRes.Containers {
			for _, res := range contRes.Resources {
				if res.Name == corev1.ResourceMemory || strings.HasPrefix(string(res.Name), corev1.ResourceHugePagesPrefix) {
					noderesourceData.updateMemoryAvailable(perNuma, res)
					continue
				}

				noderesourceData.updateAvailable(perNuma, res)
			}
		}
	}

	zones := make(topologyv1alpha1.ZoneList, 0)
	for nodeID, resList := range perNuma {
		zone := topologyv1alpha1.Zone{
			Name:      makeZoneName(nodeID),
			Type:      "Node",
			Resources: make(topologyv1alpha1.ResourceInfoList, 0),
		}

		costs, err := makeCostsPerNumaNode(noderesourceData.topo.Nodes, nodeID)
		if err != nil {
			klog.ErrorS(err, "failed to calculate costs for NUMA node", "nodeID", nodeID)
		} else {
			zone.Costs = costs
		}

		for name, resData := range resList {
			allocatableQty := *resource.NewQuantity(resData.allocatable, resource.DecimalSI)
			capacityQty := *resource.NewQuantity(resData.capacity, resource.DecimalSI)
			availableQty := *resource.NewQuantity(resData.available, resource.DecimalSI)
			zone.Resources = append(zone.Resources, topologyv1alpha1.ResourceInfo{
				Name:        name.String(),
				Available:   availableQty,
				Allocatable: allocatableQty,
				Capacity:    capacityQty,
			})
		}
		zones = append(zones, zone)
	}
	return zones
}

// NewResourcesAggregatorFromData is used to aggregate resource information based on the received data from underlying hardware and podresource API
func NewResourcesAggregatorFromData(topo *ghw.TopologyInfo, resp *podresourcesapi.AllocatableResourcesResponse, memoryResourceCapacity utils.NumaMemoryResources, excludeList ExcludeResourceList) ResourcesAggregator {
	allDevs := getContainerDevicesFromAllocatableResources(resp, topo)
	return &nodeResources{
		topo:                           topo,
		resourceID2NUMAID:              makeResourceMap(len(topo.Nodes), allDevs),
		perNUMAAllocatable:             makeNodeAllocatable(allDevs, resp.GetMemory()),
		reservedCPUIDPerNUMA:           makeReservedCPUMap(topo.Nodes, allDevs),
		memoryResourcesCapacityPerNUMA: memoryResourceCapacity,
		excludeList:                    excludeList,
	}
}

func getMemoryResourcesCapacity() (utils.NumaMemoryResources, error) {
	memoryResources, err := utils.GetNumaMemoryResources()
	if err != nil {
		return nil, err
	}

	capacity := make(utils.NumaMemoryResources)
	for numaID, resources := range memoryResources {
		if _, ok := capacity[numaID]; !ok {
			capacity[numaID] = map[corev1.ResourceName]int64{}
		}

		for resourceName, value := range resources {
			if _, ok := capacity[numaID][resourceName]; !ok {
				capacity[numaID][resourceName] = 0
			}
			capacity[numaID][resourceName] += value
		}
	}

	return capacity, nil
}

// makeZoneName returns the canonical name of a NUMA zone from its ID.
func makeZoneName(nodeID int) string {
	return fmt.Sprintf("node-%d", nodeID)
}

func findNodeByID(nodes []*ghw.TopologyNode, nodeID int) *ghw.TopologyNode {
	for _, node := range nodes {
		if node.ID == nodeID {
			return node
		}
	}
	return nil
}

// makeCostsPerNumaNode builds the cost map to reach all the known NUMA zones (mapping (numa zone) -> cost) starting from the given NUMA zone.
func makeCostsPerNumaNode(nodes []*ghw.TopologyNode, nodeIDSrc int) ([]topologyv1alpha1.CostInfo, error) {
	nodeSrc := findNodeByID(nodes, nodeIDSrc)
	if nodeSrc == nil {
		return nil, fmt.Errorf("unknown node: %d", nodeIDSrc)
	}
	nodeCosts := make([]topologyv1alpha1.CostInfo, 0)
	for nodeIDDst, dist := range nodeSrc.Distances {
		// TODO: this assumes there are no holes (= no offline node) in the distance vector
		nodeCosts = append(nodeCosts, topologyv1alpha1.CostInfo{
			Name:  makeZoneName(nodeIDDst),
			Value: int64(dist),
		})
	}
	return nodeCosts, nil
}

// makeResourceMap creates the mapping (resource name) -> (device ID) -> (NUMA node ID) from the given slice of ContainerDevices.
// this is useful to quickly learn the NUMA ID of a given (resource, device).
func makeResourceMap(numaNodes int, devices []*podresourcesapi.ContainerDevices) map[string]map[string]int {
	resourceMap := make(map[string]map[string]int)

	for _, device := range devices {
		resourceName := device.GetResourceName()
		_, ok := resourceMap[resourceName]
		if !ok {
			resourceMap[resourceName] = make(map[string]int)
		}
		for _, node := range device.GetTopology().GetNodes() {
			nodeID := int(node.GetID())
			for _, deviceID := range device.GetDeviceIds() {
				resourceMap[resourceName][deviceID] = nodeID
			}
		}
	}
	return resourceMap
}

// makeNodeAllocatable computes the node allocatable as mapping (NUMA node ID) -> Resource -> Allocatable (amount, int).
// The computation is done assuming all the resources to represent the allocatable for are represented on a slice
// of ContainerDevices. No special treatment is done for CPU IDs. See getContainerDevicesFromAllocatableResources.
func makeNodeAllocatable(devices []*podresourcesapi.ContainerDevices, memoryBlocks []*podresourcesapi.ContainerMemory) map[int]map[corev1.ResourceName]int64 {
	perNUMAAllocatable := make(map[int]map[corev1.ResourceName]int64)
	// initialize with the capacities
	for _, device := range devices {
		resourceName := device.GetResourceName()
		for _, node := range device.GetTopology().GetNodes() {
			nodeID := int(node.GetID())
			nodeRes, ok := perNUMAAllocatable[nodeID]
			if !ok {
				nodeRes = make(map[corev1.ResourceName]int64)
			}
			nodeRes[corev1.ResourceName(resourceName)] += int64(len(device.GetDeviceIds()))
			perNUMAAllocatable[nodeID] = nodeRes
		}
	}

	for _, block := range memoryBlocks {
		memoryType := corev1.ResourceName(block.GetMemoryType())

		blockTopology := block.GetTopology()
		if blockTopology == nil {
			continue
		}

		for _, node := range blockTopology.GetNodes() {
			nodeID := int(node.GetID())
			if _, ok := perNUMAAllocatable[nodeID]; !ok {
				perNUMAAllocatable[nodeID] = make(map[corev1.ResourceName]int64)
			}

			if _, ok := perNUMAAllocatable[nodeID][memoryType]; !ok {
				perNUMAAllocatable[nodeID][memoryType] = 0
			}

			// I do not like the idea to cast from uint64 to int64, but until the memory size does not go over
			// 8589934592Gi, it should be ok
			perNUMAAllocatable[nodeID][memoryType] += int64(block.GetSize_())
		}
	}

	return perNUMAAllocatable
}

func getCPUs(devices []*podresourcesapi.ContainerDevices) map[string]int {
	cpuMap := make(map[string]int)
	for _, device := range devices {
		if device.GetResourceName() == "cpu" {
			for _, devId := range device.DeviceIds {
				cpuMap[devId] = int(device.Topology.Nodes[0].ID)
			}
		}
	}
	return cpuMap
}

func makeReservedCPUMap(nodes []*ghw.TopologyNode, devices []*podresourcesapi.ContainerDevices) map[int][]string {
	reservedCPUsPerNuma := make(map[int][]string)
	cpus := getCPUs(devices)
	for _, node := range nodes {
		nodeID := node.ID
		for _, core := range node.Cores {
			for _, cpu := range core.LogicalProcessors {
				cpuID := fmt.Sprintf("%d", cpu)
				_, ok := cpus[cpuID]
				if !ok {
					cpuIDList, ok := reservedCPUsPerNuma[nodeID]
					if !ok {
						cpuIDList = make([]string, 0)
					}
					cpuIDList = append(cpuIDList, cpuID)
					reservedCPUsPerNuma[nodeID] = cpuIDList
				}
			}
		}
	}
	return reservedCPUsPerNuma
}
