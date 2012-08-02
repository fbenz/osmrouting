# Team FortyTwo ReadMe

# requirements:
# - Go Version 1
#   such that the 'go' tool is available
# - Metis
#   'gpmetis' has to be in the same folder as our 'partition' program (/bin)

# parameters:
# pbf_file: OSM PBF file
# path: absolut path to the graph dir
# port: port of the server

# build everything
./build.sh

# preprocessing
./preprocess.sh pbf_file path

# start server
./bin/server -dir path -port port
