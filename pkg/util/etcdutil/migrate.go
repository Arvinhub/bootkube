package etcdutil

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/labels"
)

func Migrate() error {
	time.Sleep(30 * time.Second)
	// TODO: poll if TPR ready?
	fmt.Println("etcd TPR is ready ===")

	kubecli, err := unversioned.New(&restclient.Config{
		Host: "http://127.0.0.1:8080",
	})
	if err != nil {
		return err
	}

	ip := ""
	for {
		podList, err := kubecli.Pods("kube-system").List(api.ListOptions{
			LabelSelector: labels.SelectorFromSet(labels.Set{"k8s-app": "boot-etcd"}),
		})
		if err != nil {
			glog.Errorf("fail to list 'boot-kube' pod, retrying: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		if len(podList.Items) < 1 {
			glog.Warningf("no 'boot-kube' pod found, retrying...")
			time.Sleep(5 * time.Second)
			continue
		}
		pod := podList.Items[0]
		ip = pod.Status.PodIP
		fmt.Println("get ip:", ip)
		if len(ip) != 0 {
			break
		}
		time.Sleep(5 * time.Second)
	}

	b := []byte(fmt.Sprintf(`{
  "apiVersion": "coreos.com/v1",
  "kind": "EtcdCluster",
  "metadata": {
    "name": "etcd-cluster",
    "namespace": "kube-system"
  },
  "spec": {
    "size": 1,
    "version": "v3.1.0-alpha.1",
    "seed": {
      "MemberClientEndpoints": [
        "http://%s:2379"
      ],
      "RemoveDelay": 60
    }
  }
}`, ip))

	resp, err := kubecli.Client.Post(
		"http://127.0.0.1:8080/apis/coreos.com/v1/namespaces/kube-system/etcdclusters",
		"application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status: %v", resp.Status)
	}

	// TODO: check new etcd cluster ready using "10.3.0.20"
	time.Sleep(600 * time.Second)

	return nil
}
