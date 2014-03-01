OSM Routing
==========

A fast and optimal routing service for cars, bikes, and pedestrians that is based on OpenStreetMap data. The project consists of a server and several programs for the preprocessing (parser, refine, partition, metric, kdtreebuilder). The service scales up to the whole planet.


Requirements
---------
* Go version 1.0 or above (tested with 1.0 and 1.2)
* Metis: 'gpmetis' has to be available for the preprocessing

Running Tests
-------------

To build everything, use:

    build.sh

To execute the preprocessing steps, use:

    preprocess.sh pbf_file path

To start up the server, use:

    bin/server -dir path -port port

The following parameters occur in the commands above:
* pbf_file: OSM PBF file
* path: absolut path to the graph dir
* port: port of the server
