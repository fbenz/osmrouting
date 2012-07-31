# requirements
# Go Version 1
# such that the 'go' tool is available

# parameters
# pbf_file: OSM PBF file
# access_type: car, bike, foot or combinations, e.g. car,bike
# path: the path to the graph that is produced by the parser
# port: port of the server

# preprocessing
parser -i=pbf_file -f=access_type
partition -dir=path
metric -dir=path
kdtreebuilder -dir=path

# start server
server -dir=path -port=port
