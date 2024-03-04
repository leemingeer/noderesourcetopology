package topologyupdater

import (
	"context"
	"fmt"
	"github.com/leemingeer/noderesourcetopology/pkg/apihelper"
	"github.com/leemingeer/noderesourcetopology/pkg/apis/topology/v1alpha1"
	"github.com/leemingeer/noderesourcetopology/pkg/podres"
	"github.com/leemingeer/noderesourcetopology/pkg/resourcemonitor"
	"github.com/leemingeer/noderesourcetopology/pkg/topology-updater/kubeletnotifier"
	"github.com/leemingeer/noderesourcetopology/pkg/topologypolicy"
	"github.com/leemingeer/noderesourcetopology/pkg/utils"
	"github.com/leemingeer/noderesourcetopology/pkg/utils/kubeconf"
	"github.com/leemingeer/noderesourcetopology/pkg/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"net/url"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/api/errors"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
	"sigs.k8s.io/yaml"
)

const (
	// TopologyManagerPolicyAttributeName represents an attribute which defines Topology Manager Policy
	TopologyManagerPolicyAttributeName = "topologyManagerPolicy"
	// TopologyManagerScopeAttributeName represents an attribute which defines Topology Manager Policy Scope
	TopologyManagerScopeAttributeName = "topologyManagerScope"
)

// Args are the command line arguments
type Args struct {
	MetricsPort     int
	NoPublish       bool
	Oneshot         bool
	KubeConfigFile  string
	ConfigFile      string
	KubeletStateDir string

	Klog map[string]*utils.KlogFlagVal
}

// NewTopologyUpdater creates a new NfdTopologyUpdater instance.
func NewTopologyUpdater(args Args, resourcemonitorArgs resourcemonitor.Args) (TopologyUpdater, error) {
	eventSource := make(chan kubeletnotifier.Info)

	ntf, err := kubeletnotifier.New(resourcemonitorArgs.SleepInterval, eventSource, args.KubeletStateDir)
	if err != nil {
		return nil, err
	}
	go ntf.Run()

	kubeletConfigFunc, err := getKubeletConfigFunc(resourcemonitorArgs.KubeletConfigURI, resourcemonitorArgs.APIAuthTokenFile)
	if err != nil {
		return nil, err
	}

	updater := &topologyUpdater{
		args:                args,
		resourcemonitorArgs: resourcemonitorArgs,
		stop:                make(chan struct{}, 1),
		nodeName:            utils.NodeName(),
		eventSource:         eventSource,
		config:              &Config{},
		kubeletConfigFunc:   kubeletConfigFunc,
	}
	if args.ConfigFile != "" {
		updater.configFilePath = filepath.Clean(args.ConfigFile)
	}
	return updater, nil
}

type TopologyUpdater interface {
	Run() error
	Stop()
}

// NFDConfig contains the configuration settings of NFDTopologyUpdater.
type Config struct {
	ExcludeList map[string][]string
}
type topologyUpdater struct {
	nodeName            string
	args                Args
	apihelper           apihelper.APIHelpers
	resourcemonitorArgs resourcemonitor.Args
	stop                chan struct{} // channel for signaling stop
	eventSource         <-chan kubeletnotifier.Info
	configFilePath      string
	config              *Config
	kubeletConfigFunc   func() (*kubeletconfigv1beta1.KubeletConfiguration, error)
}

// Run nfdTopologyUpdater. Returns if a fatal error is encountered, or, after
// one request if OneShot is set to 'true' in the updater args.
func (w *topologyUpdater) Run() error {
	klog.InfoS("Run Topology Updater", "version", version.Get(), "nodeName", w.nodeName)

	podResClient, err := podres.GetPodResClient(w.resourcemonitorArgs.PodResourceSocketPath)
	if err != nil {
		return fmt.Errorf("failed to get PodResource Client: %w", err)
	}

	kubeconfig, err := apihelper.GetKubeconfig(w.args.KubeConfigFile)
	if err != nil {
		return err
	}
	w.apihelper = apihelper.K8sHelpers{Kubeconfig: kubeconfig}

	if err := w.configure(); err != nil {
		return fmt.Errorf("faild to configure Node Feature Discovery Topology Updater: %w", err)
	}

	// Register to metrics server
	//if w.args.MetricsPort > 0 {
	//	m := utils.CreateMetricsServer(w.args.MetricsPort,
	//		buildInfo,
	//		scanErrors)
	//	go m.Run()
	//	registerVersion(version.Get())
	//	defer m.Stop()
	//}

	var resScan resourcemonitor.ResourcesScanner

	resScan, err = resourcemonitor.NewPodResourcesScanner(w.resourcemonitorArgs.Namespace, podResClient, w.apihelper, w.resourcemonitorArgs.PodSetFingerprint)
	if err != nil {
		return fmt.Errorf("failed to initialize ResourceMonitor instance: %w", err)
	}

	// CAUTION: these resources are expected to change rarely - if ever.
	// So we are intentionally do this once during the process lifecycle.
	// TODO: Obtain node resources dynamically from the podresource API
	// zonesChannel := make(chan v1alpha1.ZoneList)
	var zones v1alpha1.ZoneList

	excludeList := resourcemonitor.NewExcludeResourceList(w.config.ExcludeList, w.nodeName)
	resAggr, err := resourcemonitor.NewResourcesAggregator(podResClient, excludeList)
	if err != nil {
		return fmt.Errorf("failed to obtain node resource information: %w", err)
	}

	for {
		select {
		case info := <-w.eventSource:
			klog.V(4).InfoS("event received, scanning...", "event", info.Event)
			scanResponse, err := resScan.Scan()
			klog.V(1).InfoS("received updated pod resources", "podResources", utils.DelayedDumper(scanResponse.PodResources))
			if err != nil {
				klog.ErrorS(err, "scan failed")
				scanErrors.Inc()
				continue
			}
			zones = resAggr.Aggregate(scanResponse.PodResources)
			klog.V(1).InfoS("aggregated resources identified", "resourceZones", utils.DelayedDumper(zones))
			readKubeletConfig := false
			if info.Event == kubeletnotifier.IntervalBased {
				readKubeletConfig = true
			}

			if !w.args.NoPublish {
				if err = w.updateNodeResourceTopology(zones, scanResponse, readKubeletConfig); err != nil {
					return err
				}
			}

			if w.args.Oneshot {
				return nil
			}

		case <-w.stop:
			klog.InfoS("shutting down nfd-topology-updater")
			return nil
		}
	}
}

func (w *topologyUpdater) updateNodeResourceTopology(zoneInfo v1alpha1.ZoneList, scanResponse resourcemonitor.ScanResponse, readKubeletConfig bool) error {
	cli, err := w.apihelper.GetTopologyClient()
	if err != nil {
		return err
	}

	nrt, err := cli.TopologyV1alpha1().NodeResourceTopologies().Get(context.TODO(), w.nodeName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		nrtNew := v1alpha1.NodeResourceTopology{
			ObjectMeta: metav1.ObjectMeta{
				Name: w.nodeName,
			},
			Zones:      zoneInfo,
			Attributes: v1alpha1.AttributeList{},
		}

		if err := w.updateNRTTopologyManagerInfo(&nrtNew); err != nil {
			return err
		}

		updateAttributes(&nrtNew.Attributes, scanResponse.Attributes)

		if _, err := cli.TopologyV1alpha1().NodeResourceTopologies().Create(context.TODO(), &nrtNew, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("failed to create NodeResourceTopology: %w", err)
		}
		return nil
	} else if err != nil {
		return err
	}

	nrtMutated := nrt.DeepCopy()
	nrtMutated.Zones = zoneInfo

	attributes := scanResponse.Attributes

	if readKubeletConfig {
		if err := w.updateNRTTopologyManagerInfo(nrtMutated); err != nil {
			return err
		}
	}

	updateAttributes(&nrtMutated.Attributes, attributes)

	nrtUpdated, err := cli.TopologyV1alpha1().NodeResourceTopologies().Update(context.TODO(), nrtMutated, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update NodeResourceTopology: %w", err)
	}

	klog.V(4).InfoS("NodeResourceTopology object updated", "nodeResourceTopology", utils.DelayedDumper(nrtUpdated))
	return nil
}

// Stop NFD Topology Updater
func (w *topologyUpdater) Stop() {
	select {
	case w.stop <- struct{}{}:
	default:
	}
}

func (w *topologyUpdater) configure() error {
	if w.configFilePath == "" {
		klog.InfoS("no configuration file specified")
		return nil
	}

	b, err := os.ReadFile(w.configFilePath)
	if err != nil {
		// config is optional
		if os.IsNotExist(err) {
			klog.InfoS("configuration file not found", "path", w.configFilePath)
			return nil
		}
		return err
	}

	err = yaml.Unmarshal(b, w.config)
	if err != nil {
		return fmt.Errorf("failed to parse configuration file %q: %w", w.configFilePath, err)
	}
	klog.InfoS("configuration file parsed", "path", w.configFilePath, "config", w.config)
	return nil
}

func (w *topologyUpdater) detectTopologyPolicyAndScope() (string, string, error) {
	klConfig, err := w.kubeletConfigFunc()
	if err != nil {
		return "", "", err
	}

	return klConfig.TopologyManagerPolicy, klConfig.TopologyManagerScope, nil
}

func (w *topologyUpdater) updateNRTTopologyManagerInfo(nrt *v1alpha1.NodeResourceTopology) error {
	policy, scope, err := w.detectTopologyPolicyAndScope()
	if err != nil {
		return fmt.Errorf("failed to detect TopologyManager's policy and scope: %w", err)
	}

	tmAttributes := createTopologyAttributes(policy, scope)
	deprecatedTopologyPolicies := []string{string(topologypolicy.DetectTopologyPolicy(policy, scope))}

	updateAttributes(&nrt.Attributes, tmAttributes)
	nrt.TopologyPolicies = deprecatedTopologyPolicies

	return nil
}

func getKubeletConfigFunc(uri, apiAuthTokenFile string) (func() (*kubeletconfigv1beta1.KubeletConfiguration, error), error) {
	u, err := url.ParseRequestURI(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse -kubelet-config-uri: %w", err)
	}

	// init kubelet API client
	var klConfig *kubeletconfigv1beta1.KubeletConfiguration
	switch u.Scheme {
	case "file":
		return func() (*kubeletconfigv1beta1.KubeletConfiguration, error) {
			klConfig, err = kubeconf.GetKubeletConfigFromLocalFile(u.Path)
			if err != nil {
				return nil, fmt.Errorf("failed to read kubelet config: %w", err)
			}
			return klConfig, err
		}, nil
	case "https":
		restConfig, err := kubeconf.InsecureConfig(u.String(), apiAuthTokenFile)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize rest config for kubelet config uri: %w", err)
		}

		return func() (*kubeletconfigv1beta1.KubeletConfiguration, error) {
			klConfig, err = kubeconf.GetKubeletConfiguration(restConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to get kubelet config from configz endpoint: %w", err)
			}
			return klConfig, nil
		}, nil
	}

	return nil, fmt.Errorf("unsupported URI scheme: %v", u.Scheme)
}

func createTopologyAttributes(policy string, scope string) v1alpha1.AttributeList {
	return v1alpha1.AttributeList{
		{
			Name:  TopologyManagerPolicyAttributeName,
			Value: policy,
		},
		{
			Name:  TopologyManagerScopeAttributeName,
			Value: scope,
		},
	}
}

func updateAttributes(lhs *v1alpha1.AttributeList, rhs v1alpha1.AttributeList) {
	for _, attr := range rhs {
		updateAttribute(lhs, attr)
	}
}
func updateAttribute(attrList *v1alpha1.AttributeList, attrInfo v1alpha1.AttributeInfo) {
	if attrList == nil {
		return
	}

	for idx := range *attrList {
		if (*attrList)[idx].Name == attrInfo.Name {
			(*attrList)[idx].Value = attrInfo.Value
			return
		}
	}
	*attrList = append(*attrList, attrInfo)
}
