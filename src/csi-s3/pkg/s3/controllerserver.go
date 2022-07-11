/*
Copyright 2017 The Kubernetes Authors.

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

package s3

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/container-storage-interface/spec/lib/go/csi"
	csicommon "github.com/kubernetes-csi/drivers/pkg/csi-common"
)

type controllerServer struct {
	*csicommon.DefaultControllerServer
}

func (cs *controllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *controllerServer) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *controllerServer) ControllerGetVolume(ctx context.Context, req *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *controllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	volumeID := sanitizeVolumeID(req.GetName())
	bucketName := volumeID

	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME); err != nil {
		glog.V(3).Infof("invalid create volume req: %v", req)
		return nil, err
	}

	// Check arguments
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Name missing in request")
	}
	if req.GetVolumeCapabilities() == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume Capabilities missing in request")
	}

	//params := req.GetParameters()
	//mounter := params[mounterTypeKey]

	glog.V(4).Infof("Got a request to create volume %s", volumeID)

	s3, err := newS3ClientFromSecrets(req.GetSecrets())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize S3 client: %s", err)
	}
	if len(s3.cfg.Bucket) != 0 {
		bucketName = s3.cfg.Bucket
	}
	exists, err := s3.bucketExists(bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if bucket %s exists: %v", bucketName, err)
	}
	if exists == false {
		if s3.cfg.Provision {
			if err = s3.createBucket(bucketName); err != nil {
				return nil, fmt.Errorf("failed to create volume %s: %v", volumeID, err)
			}
		} else {
			return nil, fmt.Errorf("s3 bucket: %s does not exist", bucketName)
		}
	}
	//b := &bucket{
	//	Name:          bucketName,
	//	Mounter:       mounter,
	//}

	//TODO check for readonly

	//if err := s3.setBucket(b); err != nil {
	//	return nil, fmt.Errorf("Error setting bucket metadata: %v", err)
	//}

	glog.V(4).Infof("create volume %s", volumeID)
	s3Vol := s3Volume{}
	s3Vol.VolName = volumeID
	s3Vol.VolID = volumeID
	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      volumeID,
			CapacityBytes: 10000000000000, //TODO what policy should dictate it the capacity?
			VolumeContext: req.GetParameters(),
		},
	}, nil
}

func (cs *controllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	volumeID := req.GetVolumeId()
	bucketName := volumeID

	// Check arguments
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}

	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME); err != nil {
		glog.V(3).Infof("Invalid delete volume req: %v", req)
		return nil, err
	}
	glog.V(4).Infof("Deleting volume %s", volumeID)

	s3, err := newS3ClientFromSecrets(req.GetSecrets())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize S3 client: %s", err)
	}
	if len(s3.cfg.Bucket) != 0 {
		bucketName = s3.cfg.Bucket
	}
	exists, err := s3.bucketExists(bucketName)
	if err != nil {
		return nil, err
	}
	removeOnDelete := s3.cfg.RemoveOnDelete
	glog.V(5).Info("Remove on delete value %s", fmt.Sprint(removeOnDelete))
	if exists && removeOnDelete {
		if err := s3.removeBucket(bucketName); err != nil {
			glog.V(3).Infof("Failed to remove volume %s and bucket %s: %v", volumeID, bucketName, err)
			return nil, err
		}
	} else {
		glog.V(5).Infof("Bucket %s does not exist or not remove-on-delete flag, ignoring request", bucketName)
	}

	return &csi.DeleteVolumeResponse{}, nil
}

func (cs *controllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {

	// Check arguments
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	if req.GetVolumeCapabilities() == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities missing in request")
	}

	s3, err := newS3ClientFromSecrets(req.GetSecrets())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize S3 client: %s", err)
	}
	exists, err := s3.bucketExists(req.GetVolumeId())
	if err != nil {
		return nil, err
	}
	if !exists {
		// return an error if the volume requested does not exist
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Volume with id %s does not exist", req.GetVolumeId()))
	}

	// We currently only support RWO
	supportedAccessMode := &csi.VolumeCapability_AccessMode{
		Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
	}

	for _, cap := range req.VolumeCapabilities {
		if cap.GetAccessMode().GetMode() != supportedAccessMode.GetMode() {
			return &csi.ValidateVolumeCapabilitiesResponse{Message: "Only single node writer is supported"}, nil
		}
	}

	return &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeCapabilities: []*csi.VolumeCapability{
				{
					AccessMode: supportedAccessMode,
				},
			},
		},
	}, nil
}

func (cs *controllerServer) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	return &csi.ControllerExpandVolumeResponse{}, status.Error(codes.Unimplemented, "ControllerExpandVolume is not implemented")
}

func createVolume(volumeID string, s3args map[string]string) (*s3Volume, error) {
	glog.V(4).Infof("Got a request to create volume %s", volumeID)

	//Do not check this line in to git - remove after debug
	glog.V(4).Infof("Received request for %v", s3args)

	volID := sanitizeVolumeID(volumeID)
	provision, _ := strconv.ParseBool(s3args["provision"])

	if len(s3args["bucket"]) == 0 {
		// Only use vol id for the bucket if provision is set to true
		if provision {
			s3args["bucket"] = volID
		} else {
			return nil, fmt.Errorf("Neither existing bucket provided nor provisioning requested")
		}
	}

	s3, err := newS3ClientFromSecrets(s3args)

	if err != nil {
		glog.V(4).Infof("Could not create s3 client instance: %s", err)
		return nil, fmt.Errorf("failed to initialize S3 client: %s", err)
	} else {
		glog.V(4).Infof("Created s3 client instance ")
	}

	exists, err := s3.bucketExists(s3args["bucket"])
	if err != nil {
		glog.V(4).Infof("Could not check if bucket exists: %s", err)
		return nil, fmt.Errorf("failed to check if bucket %s exists: %v", s3args["bucket"], err)
	}

	if !exists {
		if s3.cfg.Provision {
			if err = s3.createBucket(s3args["bucket"]); err != nil {
				return nil, fmt.Errorf("failed to create volume %s: %v", volumeID, err)
			}
		} else {
			return nil, fmt.Errorf("s3 bucket: %s does not exist", s3args["bucket"])
		}
	}
	//b := &bucket{
	//	Name:          bucketName,
	//	Mounter:       mounter,
	//}

	//TODO check for readonly

	//if err := s3.setBucket(b); err != nil {
	//	return nil, fmt.Errorf("Error setting bucket metadata: %v", err)
	//}

	glog.V(4).Infof("create volume %s", volumeID)
	s3Vol := s3Volume{}
	s3Vol.VolName = volumeID
	s3Vol.VolID = volumeID
	s3Vol.Bucket = s3args["bucket"]

	return &s3Vol, nil
}

func sanitizeVolumeID(volumeID string) string {
	volumeID = strings.ToLower(volumeID)
	if len(volumeID) > 63 {
		h := sha1.New()
		io.WriteString(h, volumeID)
		volumeID = hex.EncodeToString(h.Sum(nil))
	}
	return volumeID
}
