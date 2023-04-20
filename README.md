# Golang version of arcclimate

## Quick Start


For Ubuntu/Debian user
```
sudo apt install golang # if you didnot install golang
go install github.com/udawtr/arcclimate-go@latest
~/go/bin/arcclimate-go 33.8834976 130.8751773 --mode EA -o test.csv
```

For Windows user
```
go install github.com/udawtr/arcclimate-go@latest
arcclimate-go 33.8834976 130.8751773 --mode EA -o test.csv
```

## Difference from Python version

* Run very fast. More than 10x.
* There is no control function for log output.
* For speed, mesh_3d_elevation.csv has been split into mesh_3d_ele_{mesh1d}.csv. (By split_mesh_3d_ele.py)

