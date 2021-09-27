// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package aws

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	corev1 "k8s.io/api/core/v1"
)

const (
	envVarAccessKeyID     = "AWS_ACCESS_KEY_ID"
	envVarSecretAccessKey = "AWS_SECRET_ACCESS_KEY"
	envVarRegion          = "AWS_REGION"

	dataKeyAccessKeyID     = "accessKeyID"
	dataKeySecretAccessKey = "secretAccessKey"
	dataKeyRegion          = "region"
)

type backupProvider struct {
	bucketName      string
	accessKeyID     string
	secretAccessKey string
	region          string
}

// NewBackupProvider creates a new S3 backup provider implementation.
func NewBackupProvider(credentialsData map[string]string, bucketName, region string) (*backupProvider, error) {
	accessKeyID, ok := credentialsData[dataKeyAccessKeyID]
	if !ok {
		return nil, fmt.Errorf("data map doesn't have an access key id")
	}

	secretAccessKey, ok := credentialsData[dataKeySecretAccessKey]
	if !ok {
		return nil, fmt.Errorf("data map doesn't have a secret access key")
	}

	os.Setenv(envVarAccessKeyID, accessKeyID)
	os.Setenv(envVarSecretAccessKey, secretAccessKey)

	return &backupProvider{
		bucketName:      bucketName,
		accessKeyID:     accessKeyID,
		secretAccessKey: secretAccessKey,
		region:          region,
	}, nil
}

func (b *backupProvider) CreateBucket(ctx context.Context) error {
	svc, err := b.getClient()
	if err != nil {
		return err
	}

	_, err = svc.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(b.bucketName),
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeBucketAlreadyOwnedByYou:
				return nil
			default:
				return err
			}
		} else {
			return err
		}
	}

	return svc.WaitUntilBucketExists(&s3.HeadBucketInput{
		Bucket: aws.String(b.bucketName),
	})
}

func (b *backupProvider) DeleteBucket(ctx context.Context) error {
	svc, err := b.getClient()
	if err != nil {
		return err
	}

	iter := s3manager.NewDeleteListIterator(svc, &s3.ListObjectsInput{
		Bucket: aws.String(b.bucketName),
	})

	if err := s3manager.NewBatchDeleteWithClient(svc).Delete(aws.BackgroundContext(), iter); err != nil {
		return err
	}

	_, err = svc.DeleteBucket(&s3.DeleteBucketInput{
		Bucket: aws.String(b.bucketName),
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket:
				return nil
			default:
				return err
			}
		} else {
			return err
		}
	}

	return svc.WaitUntilBucketNotExists(&s3.HeadBucketInput{
		Bucket: aws.String(b.bucketName),
	})
}

func (b *backupProvider) BucketExists(ctx context.Context) (bool, error) {
	svc, err := b.getClient()
	if err != nil {
		return false, err
	}

	result, err := svc.ListBuckets(nil)
	if err != nil {
		return false, err
	}

	for _, next := range result.Buckets {
		if aws.StringValue(next.Name) == b.bucketName {
			return true, nil
		}
	}

	return false, nil
}

func (b *backupProvider) ComputeETCDBackupConfiguration(_, etcdSecretNameBackup string) (storageProviderName string, secretData map[string][]byte, environment []corev1.EnvVar) {
	storageProviderName = "S3"

	secretData = map[string][]byte{
		dataKeyAccessKeyID:     []byte(b.accessKeyID),
		dataKeySecretAccessKey: []byte(b.secretAccessKey),
		dataKeyRegion:          []byte(b.region),
	}

	environment = []corev1.EnvVar{
		b.envVar(envVarAccessKeyID, etcdSecretNameBackup, dataKeyAccessKeyID),
		b.envVar(envVarSecretAccessKey, etcdSecretNameBackup, dataKeySecretAccessKey),
		b.envVar(envVarRegion, etcdSecretNameBackup, dataKeyRegion),
	}

	return
}

func (b *backupProvider) envVar(envVarName, etcdSecretNameBackup, dataKey string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: envVarName,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: etcdSecretNameBackup,
				},
				Key: dataKey,
			},
		},
	}
}

func (b *backupProvider) getClient() (*s3.S3, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(b.region)},
	)

	if err != nil {
		return nil, err
	}

	svc := s3.New(sess)
	return svc, nil
}
