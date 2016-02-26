#!/bin/bash -x

SCRIPT_DIR="$(dirname `ls -l $0 | awk '{ print $NF }'`)"

[[ -z $APPNAME ]] && APPNAME=pgdiff

if [[ -d bin-linux ]]; then
    tempdir="$(mktemp -d -t $APPNAME)"
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
    zip -r "${APPNAME}.zip" $APPNAME
    mv "${APPNAME}.zip" "$SCRIPT_DIR/bin-linux/"
    cd -
    echo "Built linux."
else
    echo "Skipping linux.  No bin-linux directory."
fi

#### DON'T LEAVE THIS
exit 1

if [[ -d bin-osx ]]; then
    GOOS=darwin GOARCH=386 go build -o bin-osx/${APPNAME}
    echo "Built osx32."
else
    echo "Skipping osx.  No bin-osx directory."
fi

if [[ -d bin-win ]]; then
    GOOS=windows GOARCH=386 go build -o bin-win/${APPNAME}.exe
    echo "Built win32."
else
    echo "Skipping win.  No bin-win directory."
fi
