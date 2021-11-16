/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package types

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"

	"github.com/spf13/cobra"
	clientv3 "go.etcd.io/etcd/client/v3"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/protobuf"
)

var (
	etcdCA   string
	etcdCert string
	etcdKey  string
	etcdHost string
	etcdPort string
)

func etcdClient() (*clientv3.Client, error) {

	ca, err := ioutil.ReadFile(etcdCA)
	if err != nil {
		return nil, err
	}

	keyPair, err := tls.LoadX509KeyPair(etcdCert, etcdKey)
	if err != nil {
		return nil, err
	}
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(ca)

	client, err := clientv3.New(clientv3.Config{
		Endpoints: []string{fmt.Sprintf("%s:%s", etcdHost, etcdPort)},
		TLS: &tls.Config{
			RootCAs: certPool,
			Certificates: []tls.Certificate{
				keyPair,
			},
			InsecureSkipVerify: true,
		}})
	if err != nil {
		return nil, err
	}
	return client, nil
}

// pvcCmd represents the pvc command
var PvcCmd = &cobra.Command{
	Use:   "pvc",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		fmt.Printf("etcdca: %s etcdkey: %s etcdcert: %s\n", etcdCA, etcdKey, etcdCert)
		client, err := etcdClient()
		if err != nil {
			fmt.Println(err)
			return
		}

		response, err := client.Get(context.Background(), "/registry/persistentvolumeclaims", clientv3.WithPrefix())
		if err != nil {
			fmt.Println(err)
			return
		}

		gvk := schema.GroupVersionKind{Group: v1.GroupName,
			Version: "v1", Kind: "PersistentVolumeClaim"}

		runtimeSchema := runtime.NewScheme()
		runtimeSchema.AddKnownTypeWithName(gvk, &v1.PersistentVolumeClaim{})

		protoSerializer := protobuf.NewSerializer(runtimeSchema, runtimeSchema)

		for _, rawKV := range response.Kvs {

			pvc := &v1.PersistentVolumeClaim{}
			_, _, err := protoSerializer.Decode(rawKV.Value, &gvk, pvc)
			if err != nil {
				fmt.Println(err)
				return
			}
			(*pvc).ObjectMeta.DeletionTimestamp = nil
			(*pvc).ObjectMeta.DeletionGracePeriodSeconds = nil

			var fixedPVC bytes.Buffer
			err = protoSerializer.Encode(pvc, &fixedPVC)
			if err != nil {
				fmt.Println(err)
				return
			}

			client.Put(context.Background(), fmt.Sprintf("/registry/persistentvolumeclaims/%s/%s", pvc.Namespace, pvc.Name),
				fixedPVC.String())
		}

	},
}

func init() {

	PvcCmd.Flags().StringVarP(&etcdCA, "etcdca", "c", "", "etcda")
	PvcCmd.Flags().StringVarP(&etcdCert, "etcdcert", "a", "", "etcdcert")
	PvcCmd.Flags().StringVarP(&etcdKey, "etcdkey", "k", "", "etcdkey")
	PvcCmd.Flags().StringVarP(&etcdHost, "etcdhost", "o", "localhost", "etcdhost")
	PvcCmd.Flags().StringVarP(&etcdPort, "etcdport", "p", "2379", "etcdport")
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// pvcCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// pvcCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
