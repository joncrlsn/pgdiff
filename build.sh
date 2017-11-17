#!/bin/bash
#
# Builds executable and downloadable bundle for 3 platforms
#
# For OSX and Linux:
#  * builds pgdiff 
#  * downloads pgrun 
#  * combines them, a README, and pgdiff.sh into a tgz file
#
# For Windows:
#  * builds pgdiff.exe
#  * downloads pgrun.exe
#  * combines them, a README, and pgdiff.sh into a zip file
#

SCRIPT_DIR="$(dirname `ls -l $0 | awk '{ print $NF }'`)"

[[ -z $APPNAME ]] && APPNAME=pgdiff
[[ -z $VERSION ]] && read -p "Enter version number: " VERSION

LINUX_README=README-linux.md
LINUX_FILE="${APPNAME}-linux-${VERSION}.tar.gz"

OSX_README=README-osx.md
OSX_FILE="${APPNAME}-osx-${VERSION}.tar.gz"

WIN_README=README-win.md
WIN_FILE="${APPNAME}-win-${VERSION}.zip"

if [[ -f $LINUX_README ]]; then
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
    cp "${SCRIPT_DIR}/${LINUX_README}" "$workdir/README.md"
    cd "$tempdir"
    # Make everything executable
    chmod -v ugo+x $APPNAME/*
    tarName="${tempdir}/${LINUX_FILE}"
    COPYFILE_DISABLE=true tar -cvzf "$tarName" $APPNAME
    cd -
    mv "$tarName" "${SCRIPT_DIR}/"
    echo "Built linux."
else
    echo "Skipping linux.  No $LINUX_README file."
fi

if [[ -f $OSX_README ]]; then
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
    cp "${SCRIPT_DIR}/${OSX_README}" "$workdir/README.md"
    cd "$tempdir"
    # Make everything executable
    chmod -v ugo+x $APPNAME/*
    tarName="${tempdir}/${OSX_FILE}"
    COPYFILE_DISABLE=true tar -cvzf "$tarName" $APPNAME
    cd -
    mv "$tarName" "${SCRIPT_DIR}/"
    echo "Built osx."
else
    echo "Skipping osx.  No $OSX_README file."
fi

if [[ -f $WIN_README ]]; then
    echo "  ==== Building Windows ===="
    tempdir="$(mktemp -d)"
    workdir="$tempdir/$APPNAME"
    echo $workdir
    mkdir -p $workdir
    GOOS=windows GOARCH=386 go build -o "${workdir}/${APPNAME}.exe"
    # Download pgrun to the work directory 
    # Copy the bash runtime script to the temp directory
    cp "${SCRIPT_DIR}/${WIN_README}" "$workdir/README.md"
    cd "$tempdir"
    # Make everything executable
    chmod -v ugo+x $APPNAME/*
    wget -O "${workdir}/pgrun.exe" "https://github.com/joncrlsn/pgrun/raw/master/bin-win/pgrun.exe"
    zipName="${tempdir}/${WIN_FILE}"
    zip -r "$zipName" $APPNAME
    cd -
    mv "$zipName" "${SCRIPT_DIR}/"
    echo "Built win."
else
    echo "Skipping win.  No $WIN_README file."
fi
