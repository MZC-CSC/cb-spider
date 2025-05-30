#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|nhncloud number'
        echo -e '\n\tex) '$0' aws'
        echo
        exit 0;
fi

source ../common/setup.env $1
source setup.env $1

echo -e "\n\n"
echo -e "###########################################################"
echo -e "# Try to get $1 VPC"
echo -e "###########################################################"
echo -e "\n\n"


# ex) ../common/2.vpc-get.sh aws
../common/2.vpc-get.sh $1

echo -e "\n\n"
