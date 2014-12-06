#!/bin/bash

appname=pgdiff

if [[ -d bin-linux32 ]]; then
    GOOS=linux GOARCH=386 go build -o bin-linux32/${appname}
    echo "Built linux32."
else
    echo "Skipping linux32.  No bin-linux32 directory."
fi

if [[ -d bin-linux64 ]]; then
    GOOS=linux GOARCH=amd64 go build -o bin-linux64/${appname}
    echo "Built linux64."
else
    echo "Skipping linux64.  No bin-linux64 directory."
fi

if [[ -d bin-osx32 ]]; then
    GOOS=darwin GOARCH=386 go build -o bin-osx32/${appname}
    echo "Built osx32."
else
    echo "Skipping osx32.  No bin-osx32 directory."
fi

if [[ -d bin-osx64 ]]; then
    GOOS=darwin GOARCH=amd64 go build -o bin-osx64/${appname}
    echo "Built osx64."
else
    echo "Skipping osx64.  No bin-osx64 directory."
fi

if [[ -d bin-win32 ]]; then
    GOOS=windows GOARCH=386 go build -o bin-win32/${appname}.exe
    echo "Built win32."
else
    echo "Skipping win32.  No bin-win32 directory."
fi

if [[ -d bin-win64 ]]; then
    GOOS=windows GOARCH=amd64 go build -o bin-win64/${appname}.exe
    echo "Built win64."
else
    echo "Skipping win64.  No bin-win64 directory."
fi
