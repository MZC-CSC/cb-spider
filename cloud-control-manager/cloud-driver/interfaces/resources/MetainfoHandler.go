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


type Metainfo struct {
	FID  			 string		// Function ID
}	


type AvailabilityZonesOutput struct {
	RegionName 			*string
	AvailabilityZones 	[]*AvailabilityZone
}

type AvailabilityZone struct {
	ZoneName 	*string
	State 		*string
}

type MetainfoHandler interface {
	GetAllRegionZone () ([]AvailabilityZonesOutput,error)
}
