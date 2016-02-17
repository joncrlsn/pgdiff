#!/bin/bash
#
# pgdiff.sh runs a compare on each schema type in the proper order.  At each step you are allowed to review 
# and optionally change and/or run the generated SQL.
#
# If you convert this to a windows batch file, please share it.
#
# pgdiff -U postgres -W supersecret -D maindb -O sslmode=disable -u postgres -w supersecret -d stagingdb -o sslmode=disable COLUMN
#

[[ -z $USER1 ]] && USER1=c42
[[ -z $HOST1 ]] && HOST1=localhost
[[ -z $NAME1 ]] && NAME1=cp
[[ -z $OPT1 ]]  && OPT1='sslmode=disable'

[[ -z $USER2 ]] && USER2=c42
[[ -z $HOST2 ]] && HOST2=localhost
[[ -z $NAME2 ]] && NAME2=cp-pentest
[[ -z $OPT2 ]]  && OPT2='sslmode=disable'

echo "This is the reference database:"
echo "   ${USER1}@${HOST1}/$NAME1"
read -sp "Enter DB password: " passw
PASS1=$passw
PASS2=$passw

echo
echo "This database may be changed (if you choose):"
echo "   ${USER2}@${HOST2}/$NAME2"
read -sp "Enter DB password (defaults to previous password): " passw
[[ -n $passw ]] && PASS2=$passw
echo

let i=0
function rundiff() {
    ((i++))
    local TYPE=$1
    local sqlFile="${i}-${TYPE}.sql"
    echo "Generating diff for $TYPE... $PASS1"
    ./pgdiff -U "$USER1" -W "$PASS1" -H "$HOST1" -D "$NAME1" -O "$OPT1" \
           -u "$USER2" -w "$PASS2" -h "$HOST2" -d "$NAME2" -o "$OPT2" \
           $TYPE > "$sqlFile"
    RC=$? && [[ $RC != 0 ]] && exit $RC
    echo -n "Press Enter to review the generated output: "; read x
    vi "$sqlFile"
    echo -n "Do you wish to run this against ${NAME2}? [yN]: "; read x
    if [[ $x =~ ^y ]]; then
       PGPASSWORD="$PASS2" pgrun -U $USER2 -h $HOST2 -d $NAME2 -O "$OPT2" -f "$sqlFile"
    fi
    echo
}

rundiff ROLE
rundiff SEQUENCE
rundiff TABLE
rundiff OWNER
rundiff COLUMN
rundiff INDEX
rundiff FOREIGN_KEY
rundiff GRANT_RELATIONSHIP
rundiff GRANT_ATTRIBUTE

echo "Done!"

