#!/bin/bash
#
# Builds executable and downloadable bundle for 3 platforms
#
# For OSX and Linux:
#  * builds pgdiff
#  * combines them, a README, and pgdiff.sh into a tgz file
#
# For Windows:
#  * builds pgdiff.exe
#  * combines them, a README, and pgdiff.sh into a zip file
#

SCRIPT_DIR="$(dirname `ls -l $0 | awk '{ print $NF }'`)"

[[ -z $APPNAME ]] && APPNAME=pgdiff
[[ -z $VERSION ]] && read -p "Enter version number: " VERSION

LINUX_README=README-linux.md
LINUX_FILE="${APPNAME}-linux-${VERSION}.tar.gz"

LINUX_ARM64_README=README-linux-arm64.md
LINUX_ARM64_FILE="${APPNAME}-linux-arm64-${VERSION}.tar.gz"

MACOS_ARM64_README=README-macos-arm64.md
MACOS_ARM64_FILE="${APPNAME}-macos-arm64-${VERSION}.tar.gz"

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

if [[ -f $LINUX_ARM64_README ]]; then
    echo "  ==== Building Linux-arm64 ===="
    tempdir="$(mktemp -d)"
    workdir="$tempdir/$APPNAME"
    echo $workdir
    mkdir -p $workdir
    # Build the executable
    GOOS=linux GOARCH=arm64 go build -o "$workdir/$APPNAME"
    # Copy the bash runtime script to the temp directory
    cp pgdiff.sh "$workdir/"
    cp "${SCRIPT_DIR}/${LINUX_ARM64_README}" "$workdir/README.md"
    cd "$tempdir"
    # Make everything executable
    chmod -v ugo+x $APPNAME/*
    tarName="${tempdir}/${LINUX_ARM64_FILE}"
    COPYFILE_DISABLE=true tar -cvzf "$tarName" $APPNAME
    cd -
    mv "$tarName" "${SCRIPT_DIR}/"
    echo "Built linux-arm64."
else
    echo "Skipping linux-arm64.  No $LINUX_ARM64_README file."
fi

if [[ -f $MACOS_ARM64_README ]]; then
    echo "  ==== Building macOS-arm64 ===="
    tempdir="$(mktemp -d)"
    workdir="$tempdir/$APPNAME"
    echo $workdir
    mkdir -p $workdir
    # Build the executable
    GOOS=darwin GOARCH=arm64 go build -o "$workdir/$APPNAME"
    # Copy the bash runtime script to the temp directory
    cp pgdiff.sh "$workdir/"
    cp "${SCRIPT_DIR}/${MACOS_ARM64_README}" "$workdir/README.md"
    cd "$tempdir"
    # Make everything executable
    chmod -v ugo+x $APPNAME/*
    tarName="${tempdir}/${MACOS_ARM64_FILE}"
    COPYFILE_DISABLE=true tar -cvzf "$tarName" $APPNAME
    cd -
    mv "$tarName" "${SCRIPT_DIR}/"
    echo "Built macOS-arm64."
else
    echo "Skipping macOS-arm64.  No $MACOS_ARM64_README file."
fi

if [[ -f $WIN_README ]]; then
    echo "  ==== Building Windows ===="
    tempdir="$(mktemp -d)"
    workdir="$tempdir/$APPNAME"
    echo $workdir
    mkdir -p $workdir
    GOOS=windows GOARCH=386 go build -o "${workdir}/${APPNAME}.exe"
    # Copy the bash runtime script to the temp directory
    cp "${SCRIPT_DIR}/${WIN_README}" "$workdir/README.md"
    cd "$tempdir"
    # Make everything executable
    chmod -v ugo+x $APPNAME/*
    zipName="${tempdir}/${WIN_FILE}"
    zip -r "$zipName" $APPNAME
    cd -
    mv "$zipName" "${SCRIPT_DIR}/"
    echo "Built win."
else
    echo "Skipping win.  No $WIN_README file."
fi
