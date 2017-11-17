#!/bin/bash
#
# For OSX and Linux, this script:
#  * builds pgdiff 
#  * downloads pgrun 
#  * combines them and pgdiff.sh into a tgz file
#

SCRIPT_DIR="$(dirname `ls -l $0 | awk '{ print $NF }'`)"

[[ -z $APPNAME ]] && APPNAME=pgdiff
[[ -z $VERSION ]] && read -p "Enter version number: " VERSION

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
    cp "${SCRIPT_DIR}/bin-linux/README.md" "$workdir/"
    cd "$tempdir"
    # Make everything executable
    chmod -v ugo+x $APPNAME/*
    tarName="${tempdir}/${APPNAME}-linux-${VERSION}.tar.gz"
    COPYFILE_DISABLE=true tar -cvzf "$tarName" $APPNAME
    cd -
    mv "$tarName" "${SCRIPT_DIR}/bin-linux/"
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
    cp "${SCRIPT_DIR}/bin-osx/README.md" "$workdir/"
    cd "$tempdir"
    # Make everything executable
    chmod -v ugo+x $APPNAME/*
    tarName="${tempdir}/${APPNAME}-osx-${VERSION}.tar.gz"
    COPYFILE_DISABLE=true tar -cvzf "$tarName" $APPNAME
    cd -
    mv "$tarName" "${SCRIPT_DIR}/bin-osx/"
    echo "Built osx."
else
    echo "Skipping osx.  No bin-osx directory."
fi

if [[ -d bin-win ]]; then
    echo "  ==== Building Windows ===="
    tempdir="$(mktemp -d)"
    workdir="$tempdir/$APPNAME"
    echo $workdir
    mkdir -p $workdir
    GOOS=windows GOARCH=386 go build -o "${workdir}/${APPNAME}.exe"
    # Download pgrun to the work directory 
    # Copy the bash runtime script to the temp directory
    cp "${SCRIPT_DIR}/bin-win/README.md" "$workdir/"
    cd "$tempdir"
    # Make everything executable
    chmod -v ugo+x $APPNAME/*
    wget -O "${workdir}/pgrun.exe" "https://github.com/joncrlsn/pgrun/raw/master/bin-win/pgrun.exe"
    zipName="${tempdir}/${APPNAME}-win-${VERSION}.zip"
    zip -r "$zipName" $APPNAME
    cd -
    mv "$zipName" "${SCRIPT_DIR}/bin-win/"
    echo "Built win."
else
    echo "Skipping win.  No bin-win directory."
fi
