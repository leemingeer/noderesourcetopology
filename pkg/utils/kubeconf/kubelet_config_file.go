package kubeconf

import (
	"fmt"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
	kubeletconfigscheme "k8s.io/kubernetes/pkg/kubelet/apis/config/scheme"
	"k8s.io/kubernetes/pkg/kubelet/kubeletconfig/configfiles"
	utilfs "k8s.io/kubernetes/pkg/util/filesystem"
)

func GetKubeletConfigFromLocalFile(kubeletConfigPath string) (*kubeletconfigv1beta1.KubeletConfiguration, error) {
	const errFmt = "failed to load Kubelet config file %s, error %w"

	loader, err := configfiles.NewFsLoader(&utilfs.DefaultFs{}, kubeletConfigPath)
	if err != nil {
		return nil, fmt.Errorf(errFmt, kubeletConfigPath, err)
	}

	kc, err := loader.Load()
	if err != nil {
		return nil, fmt.Errorf(errFmt, kubeletConfigPath, err)
	}

	scheme, _, err := kubeletconfigscheme.NewSchemeAndCodecs()
	if err != nil {
		return nil, err
	}

	kubeletConfig := &kubeletconfigv1beta1.KubeletConfiguration{}
	err = scheme.Convert(kc, kubeletConfig, nil)
	if err != nil {
		return nil, err
	}

	return kubeletConfig, nil
}
