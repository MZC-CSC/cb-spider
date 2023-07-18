// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, 2023

package resources

import "github.com/aws/aws-sdk-go/service/ec2"


type Metainfo struct {
	FID  			 string		// Function ID

	ZoneList 		[]KeyValue	// DiskCreating | DiskAvailable | DiskAttached | DiskDeleting | DiskError

	KeyValueList 	[]KeyValue
}	

type MetainfoHandler interface {
	GetAllRegionZone () ([]ec2.DescribeAvailabilityZonesOutput,error)
	//GetAllRegionZone(FID Metainfo) (Metainfo, error)
}
