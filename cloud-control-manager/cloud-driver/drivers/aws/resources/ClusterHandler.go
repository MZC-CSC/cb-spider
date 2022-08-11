package resources

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/iam"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"strconv"
	"strings"
)

//https://docs.aws.amazon.com/sdk-for-go/api/service/eks
//https://docs.aws.amazon.com/sdk-for-go/api/service/iam/#IAM.CreateRole
//https://docs.aws.amazon.com/sdk-for-go/api/service/iam/#IAM.GetRole
//https://docs.aws.amazon.com/sdk-for-go/api/service/eks/#EKS.CreateNodegroup

//WaitUntilAddonActive
//WaitUntilAddonDeleted
//WaitUntilClusterActive
//WaitUntilClusterDeleted
//WaitUntilFargateProfileActive
//WaitUntilFargateProfileDeleted
//WaitUntilNodegroupActive
//WaitUntilNodegroupDeleted

type AwsClusterHandler struct {
	Region idrv.RegionInfo
	Client *eks.EKS
	Iam    *iam.IAM
	//VMClient *ec2.EC2
}

type AwsNodeGroupHandler struct {
	Region   idrv.RegionInfo
	Client   *eks.EKS
	Iam      *iam.IAM
	VMClient *ec2.EC2
}

const (
	NODEGROUP_TAG string = "nodegroup"
)

//------ Cluster Management

/*
	AWS Cluster는 Role이 필수임.
	우선, roleName=spider-eks-role로 설정, 생성 시 Role의 ARN을 조회하여 사용
*/
func (ClusterHandler *AwsClusterHandler) CreateCluster(clusterReqInfo irs.ClusterInfo) (irs.ClusterInfo, error) {

	// validation check

	reqSecurityGroupIds := clusterReqInfo.Network.SecurityGroupIIDs
	var securityGroupIds []*string
	for _, securityGroupIID := range reqSecurityGroupIds {
		securityGroupIds = append(securityGroupIds, aws.String(securityGroupIID.SystemId))
	}

	reqSubnetIds := clusterReqInfo.Network.SubnetIID
	var subnetIds []*string
	for _, subnetIID := range reqSubnetIds {
		subnetIds = append(subnetIds, aws.String(subnetIID.SystemId))
	}

	//AWS의 경우 사전에 Role의 생성이 필요하며, 현재는 role 이름을 다음 이름으로 일치 시킨다.(추후 개선)
	//예시) spider-eks-role
	eksRoleName := "spider-eks-cluster-role"
	// get Role Arn
	eksRole, err := ClusterHandler.getRole(irs.IID{SystemId: eksRoleName})
	if err != nil {
		// role 은 required 임.
		return irs.ClusterInfo{}, err
	}
	roleArn := eksRole.Role.Arn

	reqK8sVersion := clusterReqInfo.Version

	// create cluster
	input := &eks.CreateClusterInput{
		Name: aws.String(clusterReqInfo.IId.SystemId),
		ResourcesVpcConfig: &eks.VpcConfigRequest{
			//SecurityGroupIds: []*string{
			//	aws.String("sg-6979fe18"),
			//},
			SecurityGroupIds: securityGroupIds,
			//SubnetIds: []*string{
			//	aws.String("subnet-6782e71e"),
			//	aws.String("subnet-e7e761ac"),
			//},
			SubnetIds: subnetIds,
		},
		//RoleArn: aws.String("arn:aws:iam::012345678910:role/eks-service-role-AWSServiceRoleForAmazonEKS-J7ONKE3BQ4PI"),
		//RoleArn: aws.String(roleArn),
		RoleArn: roleArn,
		Version: aws.String(reqK8sVersion),
	}

	result, err := ClusterHandler.Client.CreateCluster(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case eks.ErrCodeResourceInUseException:
				fmt.Println(eks.ErrCodeResourceInUseException, aerr.Error())
			case eks.ErrCodeResourceLimitExceededException:
				fmt.Println(eks.ErrCodeResourceLimitExceededException, aerr.Error())
			case eks.ErrCodeInvalidParameterException:
				fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
			case eks.ErrCodeClientException:
				fmt.Println(eks.ErrCodeClientException, aerr.Error())
			case eks.ErrCodeServerException:
				fmt.Println(eks.ErrCodeServerException, aerr.Error())
			case eks.ErrCodeServiceUnavailableException:
				fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
			case eks.ErrCodeUnsupportedAvailabilityZoneException:
				fmt.Println(eks.ErrCodeUnsupportedAvailabilityZoneException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
	}

	if cblogger.Level.String() == "debug" {
		spew.Dump(result)
	}

	//----- wait until Status=COMPLETE -----//  :  cluster describe .status 로 확인

	//clusterReqInfo.NodeGroupList
	NodeGroupHandler := AwsNodeGroupHandler{Client: ClusterHandler.Client}
	for _, nodeGroupInfo := range clusterReqInfo.NodeGroupList {
		nodeGroupCreateResult, err := NodeGroupHandler.CreateNodeGroup(nodeGroupInfo)
		if err != nil {
			// err 나면 return인가??
		}
		spew.Dump(nodeGroupCreateResult)
	}
	//clusterReqInfo.Network.VpcIID
	//clusterReqInfo.Addons

	clusterInfo, errClusterInfo := ClusterHandler.GetCluster(clusterReqInfo.IId)
	if errClusterInfo != nil {
		cblogger.Error(errClusterInfo.Error())
		return irs.ClusterInfo{}, errClusterInfo
	}
	return clusterInfo, nil
}

/*
	전체 클러스터 이름을 목록으로 가져온 후
	해당 이름에 맞는 클러스터 상세정보를 가져온다.
*/
func (ClusterHandler *AwsClusterHandler) ListCluster() ([]*irs.ClusterInfo, error) {
	//return irs.ClusterInfo{}, nil

	input := &eks.ListClustersInput{}
	result, err := ClusterHandler.Client.ListClusters(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case eks.ErrCodeInvalidParameterException:
				fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
			case eks.ErrCodeClientException:
				fmt.Println(eks.ErrCodeClientException, aerr.Error())
			case eks.ErrCodeServerException:
				fmt.Println(eks.ErrCodeServerException, aerr.Error())
			case eks.ErrCodeServiceUnavailableException:
				fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return nil, err
	}

	spew.Dump(result)
	clusterList := []*irs.ClusterInfo{}
	for _, clusterName := range result.Clusters {

		clusterInfo, err := ClusterHandler.GetCluster(irs.IID{SystemId: *clusterName})
		if err != nil {

			continue //	에러가 나면 일단 skip시킴.
		}
		clusterList = append(clusterList, &clusterInfo)

	}
	return clusterList, nil
}

func (ClusterHandler *AwsClusterHandler) GetCluster(clusterIID irs.IID) (irs.ClusterInfo, error) {
	input := &eks.DescribeClusterInput{
		Name: aws.String(clusterIID.SystemId),
	}

	result, err := ClusterHandler.Client.DescribeCluster(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case eks.ErrCodeResourceNotFoundException:
				fmt.Println(eks.ErrCodeResourceNotFoundException, aerr.Error())
			case eks.ErrCodeClientException:
				fmt.Println(eks.ErrCodeClientException, aerr.Error())
			case eks.ErrCodeServerException:
				fmt.Println(eks.ErrCodeServerException, aerr.Error())
			case eks.ErrCodeServiceUnavailableException:
				fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return irs.ClusterInfo{}, err
	}
	spew.Dump(result)
	return irs.ClusterInfo{}, nil
}

func (ClusterHandler *AwsClusterHandler) DeleteCluster(clusterIID irs.IID) (bool, error) {
	input := &eks.DeleteClusterInput{
		Name: aws.String(clusterIID.SystemId),
	}

	result, err := ClusterHandler.Client.DeleteCluster(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case eks.ErrCodeResourceInUseException:
				fmt.Println(eks.ErrCodeResourceInUseException, aerr.Error())
			case eks.ErrCodeResourceNotFoundException:
				fmt.Println(eks.ErrCodeResourceNotFoundException, aerr.Error())
			case eks.ErrCodeClientException:
				fmt.Println(eks.ErrCodeClientException, aerr.Error())
			case eks.ErrCodeServerException:
				fmt.Println(eks.ErrCodeServerException, aerr.Error())
			case eks.ErrCodeServiceUnavailableException:
				fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return false, nil
	}
	spew.Dump(result)
	waitInput := &eks.DescribeClusterInput{
		Name: aws.String(clusterIID.SystemId),
	}
	err = ClusterHandler.Client.WaitUntilClusterDeleted(waitInput)
	if err != nil {
		return false, err
	}
	return true, nil
}

/*
	CreateNodeGroup 호출
*/
func (ClusterHandler *AwsClusterHandler) AddNodeGroup(clusterIID irs.IID, nodeGroup irs.IID) (irs.ClusterInfo, error) {
	return irs.ClusterInfo{}, nil
}

/*
	DeleteNodeGroup 호출
*/
func (ClusterHandler *AwsClusterHandler) RemoveNodeGroup(clusterIID irs.IID, nodeGroup irs.IID) (bool, error) {
	return false, nil
}

// Upgrade K8s version
func (ClusterHandler *AwsClusterHandler) UpgradeCluster(clusterIID irs.IID, newVersion string) (irs.ClusterInfo, error) {

	// -- version 만 update인 경우
	input := &eks.UpdateClusterVersionInput{
		Name:    aws.String(clusterIID.SystemId),
		Version: aws.String(newVersion),
	}
	result, err := ClusterHandler.Client.UpdateClusterVersion(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case eks.ErrCodeInvalidParameterException:
				fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
			case eks.ErrCodeClientException:
				fmt.Println(eks.ErrCodeClientException, aerr.Error())
			case eks.ErrCodeResourceNotFoundException:
				fmt.Println(eks.ErrCodeResourceNotFoundException, aerr.Error())
			case eks.ErrCodeServerException:
				fmt.Println(eks.ErrCodeServerException, aerr.Error())
			case eks.ErrCodeInvalidRequestException:
				fmt.Println(eks.ErrCodeInvalidRequestException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
	}
	spew.Dump(result)
	// getClusterInfo
	return irs.ClusterInfo{}, nil

	//vpcConfig := &eks.VpcConfigRequest{
	//	EndpointPrivateAccess: false,
	//	EndpointPublicAccess: false
	//	PublicAccessCidrs []*string
	//	SecurityGroupIds []*string
	//	SubnetIds []*string
	//}
	//input := &eks.UpdateClusterConfigInput{
	//	Name: aws.String(clusterIID.SystemId),
	//
	//	//ResourcesVpcConfig
	//}
	//result, err := ClusterHandler.Client.UpdateClusterConfig(input)
	//if err != nil {
	//	if aerr, ok := err.(awserr.Error); ok {
	//		switch aerr.Code() {
	//		case eks.ErrCodeInvalidParameterException:
	//			fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
	//		case eks.ErrCodeClientException:
	//			fmt.Println(eks.ErrCodeClientException, aerr.Error())
	//		case eks.ErrCodeServerException:
	//			fmt.Println(eks.ErrCodeServerException, aerr.Error())
	//		case eks.ErrCodeResourceInUseException:
	//			fmt.Println(eks.ErrCodeResourceInUseException, aerr.Error())
	//		case eks.ErrCodeResourceNotFoundException:
	//			fmt.Println(eks.ErrCodeResourceNotFoundException, aerr.Error())
	//		case eks.ErrCodeInvalidRequestException:
	//			fmt.Println(eks.ErrCodeInvalidRequestException, aerr.Error())
	//		default:
	//			fmt.Println(aerr.Error())
	//		}
	//	} else {
	//		// Print the error, cast err to awserr.Error to get the Code and
	//		// Message from an error.
	//		fmt.Println(err.Error())
	//	}
	//	return irs.ClusterInfo{}, err
	//}
	//return irs.ClusterInfo{}, nil
}

//////-------- NodeGroup API

/*
	Spider는 Group이 대문자 : NodeGroup, AWS는 소문자 : Nodegroup
*/
func (NodeGroupHandler *AwsNodeGroupHandler) CreateNodeGroup(nodeGroupReqInfo irs.NodeGroupInfo) (irs.NodeGroupInfo, error) {

	// validation check
	if nodeGroupReqInfo.MaxNumberNodes < 1 { // nodeGroupReqInfo.MaxNumberNodes 는 최소가 1이다.
		return irs.NodeGroupInfo{}, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "max는 최소가 1이다.", nil)
	}

	//IId		IID 	// {NameId, SystemId}
	//
	//// VM config.
	//ImageIID	IID
	//VMSpecName 	string
	//RootDiskType    string  // "SSD(gp2)", "Premium SSD", ...
	//RootDiskSize 	string  // "", "default", "50", "1000" (GB)
	//KeyPairIID 	IID
	//
	//// Auto Scaling config.
	//AutoScaling		bool
	//MinNumberNodes		int
	//MaxNumberNodes		int
	//
	//DesiredNumberNodes	int
	//
	//NodeList	[]IID
	//KeyValueList []KeyValue

	//eksRoleName := "spider-eks-nodegroup-role"
	//eksRoleName := "cb-eks-nodegroup-role"
	eksRoleName := "arn:aws:iam::050864702683:role/cb-eks-nodegroup-role"

	//reqSubnetIds := nodeGroupReqInfo.Network.SubnetIID
	var subnetIds []*string                                      // 없는 경우 cluster에 등록 된 subnet을 가져와서 등록할 수 있다.
	subnetIds = append(subnetIds, aws.String("subnet-262d6d7a")) //subnet-2625657a
	subnetIds = append(subnetIds, aws.String("subnet-d0ee6fab")) //subnet-d0ee6fab
	subnetIds = append(subnetIds, aws.String("subnet-875a62cb")) //subnet-875a62cb
	subnetIds = append(subnetIds, aws.String("subnet-e08f5b8b")) //subnet-e08f5b8b

	//for _, subnetIID := range reqSubnetIds {
	//	subnetIds = append(subnetIds, aws.String(subnetIID.SystemId))
	//}

	// access key
	sshKeyIID := nodeGroupReqInfo.KeyPairIID
	sshKeyIID.SystemId = "cb-webtool"
	// security group
	securityGroups := []*string{}
	securityGroups = append(securityGroups, aws.String("sg-04607666"))

	tags := map[string]string{}
	tags["key"] = NODEGROUP_TAG
	tags["value"] = nodeGroupReqInfo.IId.NameId
	//NodegroupScalingConfig
	input := &eks.CreateNodegroupInput{
		//AmiType: "", // Valid Values: AL2_x86_64 | AL2_x86_64_GPU | AL2_ARM_64 | CUSTOM | BOTTLEROCKET_ARM_64 | BOTTLEROCKET_x86_64, Required: No
		//CapacityType: aws.String("ON_DEMAND"),//Valid Values: ON_DEMAND | SPOT, Required: No

		ClusterName:   aws.String("cb-eks-cluster"),              //uri, required
		NodegroupName: aws.String(nodeGroupReqInfo.IId.SystemId), // required
		Tags:          aws.StringMap(tags),
		NodeRole:      aws.String(eksRoleName), // roleName, required
		ScalingConfig: &eks.NodegroupScalingConfig{
			DesiredSize: aws.Int64(int64(nodeGroupReqInfo.DesiredNumberNodes)),
			MaxSize:     aws.Int64(int64(nodeGroupReqInfo.MaxNumberNodes)),
			MinSize:     aws.Int64(int64(nodeGroupReqInfo.MinNumberNodes)),
		},
		Subnets: subnetIds,

		//DiskSize: 0,
		//InstanceTypes: ["",""],
		//Labels : {"key": "value"},
		//LaunchTemplate: {
		//	Id: "",
		//	Name: "",
		//	Version: ""
		//},

		//ReleaseVersion: "",
		RemoteAccess: &eks.RemoteAccessConfig{
			Ec2SshKey:            &sshKeyIID.SystemId,
			SourceSecurityGroups: securityGroups,
		},

		//Taints: [{
		//	Effect:"",
		//	Key : "",
		//	Value :""
		//}],
		//UpdateConfig: {
		//	MaxUnavailable: 0,
		//	MaxUnavailablePercentage: 0
		//},
		//Version: ""
	}

	// 필수 외에 넣을 항목들 set
	rootDiskSize, _ := strconv.ParseInt(nodeGroupReqInfo.RootDiskSize, 10, 64)
	if rootDiskSize > 0 {
		input.DiskSize = aws.Int64(rootDiskSize)
	}

	if !strings.EqualFold(nodeGroupReqInfo.VMSpecName, "") {
		var nodeSpec []string
		nodeSpec = append(nodeSpec, nodeGroupReqInfo.VMSpecName) //"p2.xlarge"
		input.InstanceTypes = aws.StringSlice(nodeSpec)
	}

	result, err := NodeGroupHandler.Client.CreateNodegroup(input) // 비동기
	if err != nil {
		return irs.NodeGroupInfo{}, err
	}

	spew.Dump(result)

	nodeGroup, err := NodeGroupHandler.GetNodeGroup(nodeGroupReqInfo.IId)
	if err != nil {
		return irs.NodeGroupInfo{}, err
	}
	return nodeGroup, nil
}

func (NodeGroupHandler *AwsNodeGroupHandler) ListNodeGroup(clusterIID irs.IID) ([]*irs.NodeGroupInfo, error) {
	input := &eks.ListNodegroupsInput{
		ClusterName: aws.String(clusterIID.SystemId),
	}
	spew.Dump(input)

	result, err := NodeGroupHandler.Client.ListNodegroups(input)
	if err != nil {
		return nil, err
	}
	spew.Dump(result)
	nodeGroupInfoList := []*irs.NodeGroupInfo{}
	for _, nodeGroupName := range result.Nodegroups {
		nodeGroupInfo, err := NodeGroupHandler.GetNodeGroup(irs.IID{SystemId: *nodeGroupName})
		if err != nil {
			//return nil, err
			continue
		}
		nodeGroupInfoList = append(nodeGroupInfoList, &nodeGroupInfo)
	}
	return nodeGroupInfoList, nil
}

/*
	node 의 instance 정보는 가져오지 않음
	Health에서 문제있는 node 정보만 가져 옴.

*/
func (NodeGroupHandler *AwsNodeGroupHandler) GetNodeGroup(nodeGroupIID irs.IID) (irs.NodeGroupInfo, error) {
	input := &eks.DescribeNodegroupInput{
		//AmiType: "", // Valid Values: AL2_x86_64 | AL2_x86_64_GPU | AL2_ARM_64 | CUSTOM | BOTTLEROCKET_ARM_64 | BOTTLEROCKET_x86_64, Required: No
		//CapacityType: aws.String("ON_DEMAND"),//Valid Values: ON_DEMAND | SPOT, Required: No

		ClusterName:   aws.String("cb-eks-cluster"),      //required
		NodegroupName: aws.String(nodeGroupIID.SystemId), // required
	}
	spew.Dump(input)

	result, err := NodeGroupHandler.Client.DescribeNodegroup(input)
	if err != nil {
		return irs.NodeGroupInfo{}, err
	}

	nodeGroupInfo, err := NodeGroupHandler.convertNodeGroup(result)
	if err != nil {
		return irs.NodeGroupInfo{}, err
	}
	return nodeGroupInfo, nil
}

func (NodeGroupHandler *AwsNodeGroupHandler) DeleteNodeGroup(nodeGroupIID irs.IID) (bool, error) {
	return false, nil
}

func (NodeGroupHandler *AwsNodeGroupHandler) AddNodes(nodeGroupIID irs.IID, number int) (irs.NodeGroupInfo, error) {
	return irs.NodeGroupInfo{}, nil
}

func (NodeGroupHandler *AwsNodeGroupHandler) RemoveNodes(nodeGroupIID irs.IID, vmIIDs *[]irs.IID) (bool, error) {
	return false, nil
}

//// create Role : role 생성까지 할 것인가?
//func (ClusterHandler *AwsClusterHandler) createRole(rule irs.IID) (irs.IID, error) {
//	result, err := ClusterHandler.Iam.CreateCluster(input)
//	input := &iam.CreateRoleInput{
//		AssumeRolePolicyDocument: aws.String("<Stringified-JSON>"),
//		Path:                     aws.String("/"),
//		RoleName:                 aws.String("Test-Role"),
//	}
//
//	result, err := ClusterHandler.Iam.CreateRole(input)
//	if err != nil {
//		if aerr, ok := err.(awserr.Error); ok {
//			switch aerr.Code() {
//			case iam.ErrCodeLimitExceededException:
//				fmt.Println(iam.ErrCodeLimitExceededException, aerr.Error())
//			case iam.ErrCodeInvalidInputException:
//				fmt.Println(iam.ErrCodeInvalidInputException, aerr.Error())
//			case iam.ErrCodeEntityAlreadyExistsException:
//				fmt.Println(iam.ErrCodeEntityAlreadyExistsException, aerr.Error())
//			case iam.ErrCodeMalformedPolicyDocumentException:
//				fmt.Println(iam.ErrCodeMalformedPolicyDocumentException, aerr.Error())
//			case iam.ErrCodeConcurrentModificationException:
//				fmt.Println(iam.ErrCodeConcurrentModificationException, aerr.Error())
//			case iam.ErrCodeServiceFailureException:
//				fmt.Println(iam.ErrCodeServiceFailureException, aerr.Error())
//			default:
//				fmt.Println(aerr.Error())
//			}
//		} else {
//			// Print the error, cast err to awserr.Error to get the Code and
//			// Message from an error.
//			fmt.Println(err.Error())
//		}
//		return
//	}
//	return irs.IID{}, nil
//}

//
func (ClusterHandler *AwsClusterHandler) getRole(role irs.IID) (*iam.GetRoleOutput, error) {
	input := &iam.GetRoleInput{
		RoleName: aws.String(role.SystemId),
	}

	result, err := ClusterHandler.Iam.GetRole(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				fmt.Println(iam.ErrCodeNoSuchEntityException, aerr.Error())
			case iam.ErrCodeServiceFailureException:
				fmt.Println(iam.ErrCodeServiceFailureException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return nil, err
	}

	return result, nil
}

//extractNodeGroup
/*
	EKS의 NodeGroup정보를 Spider의 NodeGroup으로 변경
*/
func (NodeGroupHandler *AwsNodeGroupHandler) convertNodeGroup(nodeGroupOutput *eks.DescribeNodegroupOutput) (irs.NodeGroupInfo, error) {

	nodeGroupInfo := irs.NodeGroupInfo{}

	printToJson(nodeGroupOutput)

	nodeGroup := nodeGroupOutput.Nodegroup
	//nodeRole := nodeGroup.NodeRole
	//version := nodeGroup.Version
	//releaseVersion := nodeGroup.ReleaseVersion

	//subnetList := nodeGroup.Subnets
	//nodeGroupStatus := nodeGroup.Status
	instanceTypeList := nodeGroup.InstanceTypes // spec

	//nodes := nodeGroup.Health.Issues[0].ResourceIds // 문제 있는 node들만 있는것이 아닌지..
	rootDiskSize := nodeGroup.DiskSize
	//nodeGroup.Taints// 미사용
	nodeGroupTagList := nodeGroup.Tags
	scalingConfig := nodeGroup.ScalingConfig
	//nodeGroup.RemoteAccess
	nodeGroupName := nodeGroup.NodegroupName

	//nodeGroup.LaunchTemplate //미사용
	//clusterName := nodeGroup.ClusterName
	//capacityType := nodeGroup.CapacityType // "ON_DEMAND"
	//amiType := nodeGroup.AmiType	// AL2_x86_64"
	//createTime := nodeGroup.CreatedAt
	//health := nodeGroup.Health // Code, Message, ResourceIds	// ,"Health":{"Issues":[{"Code":"NodeCreationFailure","Message":"Unhealthy nodes in the kubernetes cluster",
	//labelList := nodeGroup.Labels
	//nodeGroupArn := nodeGroup.NodegroupArn
	//nodeGroupResources := nodeGroup.Resources
	//nodeGroupResources.AutoScalingGroups// 미사용
	//nodeGroupResources.RemoteAccessSecurityGroup// 미사용

	nodes := []irs.IID{}
	for _, issue := range nodeGroup.Health.Issues {
		resourceIds := issue.ResourceIds
		for _, resourceId := range resourceIds {
			nodes = append(nodes, irs.IID{SystemId: *resourceId})
		}
	}

	nodeGroupInfo.NodeList = nodes
	nodeGroupInfo.MaxNumberNodes = int(*scalingConfig.MaxSize)
	nodeGroupInfo.MinNumberNodes = int(*scalingConfig.MinSize)

	if nodeGroupTagList == nil {
		nodeGroupTagList[NODEGROUP_TAG] = nodeGroupName // 값이없으면 nodeGroupName이랑 같은값으로 set.
	}
	nodeGroupTag := ""
	for key, val := range nodeGroupTagList {
		if strings.EqualFold("key", NODEGROUP_TAG) {
			nodeGroupTag = *val
			break
		}
		cblogger.Info(key, *val)
	}
	//printToJson(nodeGroupTagList)
	cblogger.Info("nodeGroupName=", *nodeGroupName)
	cblogger.Info("tag=", nodeGroupTagList[NODEGROUP_TAG])
	nodeGroupInfo.IId = irs.IID{
		NameId:   nodeGroupTag, // TAG에 이름
		SystemId: *nodeGroupName,
	}
	nodeGroupInfo.VMSpecName = *instanceTypeList[0]
	//nodeGroupInfo.ImageIID
	//nodeGroupInfo.KeyPairIID // keypair setting 해야하네?
	//nodeGroupInfo.RootDiskSize = strconv.FormatInt(*nodeGroup.DiskSize, 10)
	nodeGroupInfo.RootDiskSize = strconv.FormatInt(*rootDiskSize, 10)

	// TODO : node 목록 NodegroupArn 으로 조회해야하나??
	nodeList := []irs.IID{}
	//if nodeList != nil {
	//	for _, nodeId := range nodes {
	//		nodeList = append(nodeList, irs.IID{NameId: "", SystemId: *nodeId})
	//	}
	//}
	nodeGroupInfo.NodeList = nodeList
	cblogger.Info("NodeGroup")
	//	{"Nodegroup":
	//		{"AmiType":"AL2_x86_64"
	//		,"CapacityType":"ON_DEMAND"
	//		,"ClusterName":"cb-eks-cluster"
	//		,"CreatedAt":"2022-08-05T01:51:49.673Z"
	//		,"DiskSize":20
	//		,"Health":{
	//					"Issues":[
	//							{"Code":"NodeCreationFailure"
	//							,"Message":"Unhealthy nodes in the kubernetes cluster"
	//							,"ResourceIds":["i-06ee95583f3f7de5c","i-0a283a92dcce27aa8"]}]},
	//		"InstanceTypes":["t3.medium"],
	//		"Labels":{},
	//		"LaunchTemplate":null,
	//		"ModifiedAt":"2022-08-05T02:15:14.308Z",
	//		"NodeRole":"arn:aws:iam::050864702683:role/cb-eks-nodegroup-role",
	//		"NodegroupArn":"arn:aws:eks:ap-northeast-2:050864702683:nodegroup/cb-eks-cluster/cb-eks-nodegroup-test/fec135d9-c812-8862-e3b0-7b773ce70d2e","NodegroupName":"cb-eks-nodegro
	//up-test",
	//		"ReleaseVersion":"1.22.9-20220725",
	//		"RemoteAccess":{"Ec2SshKey":"cb-webtool","SourceSecurityGroups":["sg-04607666"]},
	//		"Resources":{"AutoScalingGroups":[{"Name":"eks-cb-eks-nodegroup-test-fec135d9-c812-8862-e3b0-7b773ce70d2e"}],
	//		"RemoteAccessSecurityGroup":null},
	//		"ScalingConfig":{"DesiredSize":2,"MaxSize":2,"MinSize":2},
	//		"Status":"CREATE_FAILED",
	//		"Subnets":["subnet-262d6d7a","subnet-d0ee6fab","subnet-875a62cb","subnet-e08f5b8b"],
	//		"Tags":{},
	//		"Taints":null,
	//		"UpdateConfig":{"MaxUnavailable":1,"MaxUnavailablePercentage":null},
	//		"Version":"1.22"}}

	//nodeGroupArn
	// arn format
	//arn:partition:service:region:account-id:resource-id
	//arn:partition:service:region:account-id:resource-type/resource-id
	//arn:partition:service:region:account-id:resource-type:resource-id

	printToJson(nodeGroupInfo)
	//return irs.NodeGroupInfo{}, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "추출 오류", nil)
	return nodeGroupInfo, nil
}

//-------------- Add on

func (ClusterHandler *AwsClusterHandler) CreateAddon(clusterReqInfo irs.ClusterInfo) (irs.ClusterInfo, error) {
	addOnList := clusterReqInfo.Addons
	for _, addOn := range addOnList.KeyValueList {
		cblogger.Info(addOn)
		input := &eks.CreateAddonInput{
			//AddonName: aws.String(addOn["key"]),
		}

		result, err := ClusterHandler.Client.CreateAddon(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case iam.ErrCodeNoSuchEntityException:
					fmt.Println(iam.ErrCodeNoSuchEntityException, aerr.Error())
				case iam.ErrCodeServiceFailureException:
					fmt.Println(iam.ErrCodeServiceFailureException, aerr.Error())
				default:
					fmt.Println(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				fmt.Println(err.Error())
			}
			return irs.ClusterInfo{}, err
		}
		spew.Dump(input)
	}
	return irs.ClusterInfo{}, nil
}

// toString 용
func printToJson(class interface{}) {
	e, err := json.Marshal(class)
	if err != nil {
		cblogger.Info(err)
	}
	cblogger.Info(string(e))
}
