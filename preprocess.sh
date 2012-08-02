#!/bin/bash
# $1 PBF file
# $2 Output dir
# Use absolut paths
BASEDIR="`pwd`"
echo $BASEDIR
echo $SCRIPTPATH
echo Input:  $1
echo Output: $2
mkdir -p $2-full
mkdir -p $2
cd $2-full
"$BASEDIR"/bin/parser -i $1 -f car,bike,foot
cd "$BASEDIR"/bin
./refine -i $2-full -o $2
./partition -dir $2 -uexp 15
./metric -dir $2
./kdtreebuilder -dir $2
