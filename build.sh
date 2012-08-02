#!/bin/bash
export GOPATH=`pwd`
go install parser
go install refine
go install partition
go install metric
go install kdtreebuilder
go install server
go install drawcluster
