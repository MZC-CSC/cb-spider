package resources

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
)

type AwsMetainfoHandler struct {
	Region         idrv.RegionInfo
	Client         *ec2.EC2
}

func (metaInfoHandler *AwsMetainfoHandler) GetAllRegionZone () ([]*ec2.DescribeAvailabilityZonesOutput,error){
	cblogger.Info("AWS Driver: called Metainfo()/GetAllRegionZone()!")

	hiscallInfo := GetCallLogScheme(metaInfoHandler.Region, call.METAINFO, "Meta", "GetAllRegionZone()")
	start := call.Start()

	Regionsinput := &ec2.DescribeRegionsInput{
		AllRegions : aws.Bool(true),
	}

	


	regionreq, regionresp := metaInfoHandler.Client.DescribeRegionsRequest(Regionsinput)
	regionerr := regionreq.Send()
	if regionerr != nil {
		return nil, regionerr
	}
	fmt.Print(regionresp.Regions)

	var availabilityZones []*ec2.DescribeAvailabilityZonesOutput
	for _, region := range regionresp.Regions {
		Zoneinput := &ec2.DescribeAvailabilityZonesInput{
			AllAvailabilityZones : aws.Bool(true),
		}
		metaInfoHandler.Client.Client.Config.Region = region.RegionName

		Zonereq, Zoneresp := metaInfoHandler.Client.DescribeAvailabilityZonesRequest(Zoneinput)
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		Zoneerr := Zonereq.Send()
		if Zoneerr != nil {
			return nil, Zoneerr
		}
		availabilityZones = append(availabilityZones, Zoneresp)
	}




	calllogger.Info(call.String(hiscallInfo))
	//calllogger.Info(call.String(resp))
	return availabilityZones, nil
}
