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
echo -e "# Try to list $1 VM Status"
echo -e "###########################################################"
echo -e "\n\n"


../common/5.vm-liststatus.sh $1

echo -e "\n\n"

