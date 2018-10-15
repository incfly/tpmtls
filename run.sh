#/bin/bash

git pull origin perf
go build github.com/incfly/tpmtls
sudo ./tpmtls