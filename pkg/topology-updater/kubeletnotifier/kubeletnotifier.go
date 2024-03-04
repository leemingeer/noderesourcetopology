package kubeletnotifier

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"path"
	"time"
)

type EventType string

const (
	IntervalBased EventType = "intervalBased"
	FSUpdate      EventType = "fsUpdate"

	devicePluginsDirName = "device-plugins"
)

var stateFiles = sets.NewString(
	"cpu_manager_state",
	"memory_manager_state",
	"kubelet_internal_checkpoint",
)

type Info struct {
	Event EventType
}

type Notifier struct {
	sleepInterval time.Duration
	// destination where notifications are sent
	dest    chan<- Info
	fsEvent <-chan fsnotify.Event
}

func New(sleepInterval time.Duration, dest chan<- Info, kubeletStateDir string) (*Notifier, error) {
	notif := Notifier{
		sleepInterval: sleepInterval,
		dest:          dest,
	}

	if kubeletStateDir != "" {
		devicePluginsDir := path.Join(kubeletStateDir, devicePluginsDirName)
		ch, err := createFSWatcherEvent([]string{kubeletStateDir, devicePluginsDir})
		if err != nil {
			return nil, err
		}
		notif.fsEvent = ch
	}

	return &notif, nil
}

func (n *Notifier) Run() {
	var timeEvents <-chan time.Time

	if n.sleepInterval > 0 {
		ticker := time.NewTicker(n.sleepInterval)
		defer ticker.Stop()
		timeEvents = ticker.C
	}

	// it's safe to keep the channels we don't need nil:
	// https://dave.cheney.net/2014/03/19/channel-axioms
	// "A receive from a nil channel blocks forever"
	for {
		select {
		case <-timeEvents:
			klog.V(5).InfoS("timer update received")
			i := Info{Event: IntervalBased}
			n.dest <- i

		case e := <-n.fsEvent:
			basename := path.Base(e.Name)
			klog.V(5).InfoS("fsnotify event received", "filename", basename, "op", e.Op)
			if stateFiles.Has(basename) {
				i := Info{Event: FSUpdate}
				n.dest <- i
			}
		}
	}
}

func createFSWatcherEvent(fsWatchPaths []string) (chan fsnotify.Event, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	for _, path := range fsWatchPaths {
		if err = fsWatcher.Add(path); err != nil {
			return nil, fmt.Errorf("failed to watch: %q; %w", path, err)
		}
	}
	return fsWatcher.Events, nil
}
