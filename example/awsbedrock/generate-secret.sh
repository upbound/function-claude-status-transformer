#!/usr/bin/env bash

# Helper script for creating an incluster basic creds secret.

# retrieve profile
_aws_profile=${AWS_PROFILE:=default}
# define local variables with the corresponding credentials
_aws_access_key_id=$(aws configure get aws_access_key_id --profile $_aws_profile)
_aws_secret_access_key=$(aws configure get aws_secret_access_key --profile $_aws_profile)
# setup credentials file with above credentials as the default profile.
cat <<EOF | > aws-credentials.ini 
[default]
aws_access_key_id = ${_aws_access_key_id}
aws_secret_access_key = ${_aws_secret_access_key}
EOF
# create secret in the cluster.
kubectl -n crossplane-system create secret generic example-aws-creds --from-file=credentials=aws-credentials.ini