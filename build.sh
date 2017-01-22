#!/bin/bash
#
# For OSX and Linux, this script:
#  * builds pgdiff 
#  * downloads pgrun 
#  * combines them and pgdiff.sh into a tgz file
#

SCRIPT_DIR="$(dirname `ls -l $0 | awk '{ print $NF }'`)"

[[ -z $APPNAME ]] && APPNAME=pgdiff

if [[ -d bin-linux ]]; then
    echo "  ==== Building Linux ===="
    tempdir="$(mktemp -d)"
    workdir="$tempdir/$APPNAME"
    echo $workdir
    mkdir -p $workdir
    # Build the executable
    GOOS=linux GOARCH=386 go build -o "$workdir/$APPNAME"
    # Download pgrun to the temp directory 
    wget -O "$workdir/pgrun" "https://github.com/joncrlsn/pgrun/raw/master/bin-linux/pgrun"
    # Copy the bash runtime script to the temp directory
    cp pgdiff.sh "$workdir/"
    cd "$tempdir"
    # Make everything executable
    chmod -v ugo+x $APPNAME/*
    COPYFILE_DISABLE=true tar -cvzf "${APPNAME}.tgz" $APPNAME
    cd -
    mv "${tempdir}/${APPNAME}.tgz" "${SCRIPT_DIR}/bin-linux/"
    echo "Built linux."
else
    echo "Skipping linux.  No bin-linux directory."
fi

if [[ -d bin-osx ]]; then
    echo "  ==== Building OSX ===="
    tempdir="$(mktemp -d)"
    workdir="$tempdir/$APPNAME"
    echo $workdir
    mkdir -p $workdir
    # Build the executable
    GOOS=darwin GOARCH=386 go build -o "$workdir/$APPNAME"
    # Download pgrun to the work directory 
    wget -O "$workdir/pgrun" "https://github.com/joncrlsn/pgrun/raw/master/bin-osx/pgrun"
    # Copy the bash runtime script to the temp directory
    cp pgdiff.sh "$workdir/"
    cd "$tempdir"
    # Make everything executable
    chmod -v ugo+x $APPNAME/*
    COPYFILE_DISABLE=true tar -cvzf "${APPNAME}.tgz" $APPNAME
    cd -
    mv "${tempdir}/${APPNAME}.tgz" "${SCRIPT_DIR}/bin-osx/"
    echo "Built osx."
else
    echo "Skipping osx.  No bin-osx directory."
fi

if [[ -d bin-win ]]; then
    echo "  ==== Building Windows ===="
    GOOS=windows GOARCH=386 go build -o bin-win/${APPNAME}.exe
    echo "Built win."
else
    echo "Skipping win.  No bin-win directory."
fi
