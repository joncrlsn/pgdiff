#!/bin/bash
#
# pgdiff -U1 c42 -pw1 c422006 -d1 prd-cpc -o1 sslmode=disable -U2 c42 -pw2 c422006 -d2 cp_staging -o2 sslmode=disable COLUMN

USER1=c42
HOST1=dbwan
NAME1=crashplan
OPT1=

USER2=c42
HOST2=fkd-msp
NAME2=cp_staging
OPT2=

echo -n "Enter password: "; read passw
PASS1=$passw
PASS2=$passw

function rundiff() {
    local TYPE=$1
    echo "Generating diff for $TYPE..."
    pgdiff -U1 $USER1 -pw1 $PASS1 -h1 $HOST1 -d1 $NAME1 -o1 "$OPT1" -U2 $USER2 -pw2 $PASS2 -h2 $HOST2 -d2 $NAME2 -o2 "$OPT2" $TYPE > "${TYPE}.sql"
    RC=$? && [[ $RC != 0 ]] && exit $RC
    echo -n "Press Enter to review the generated output: "; read x
    vi "${TYPE}.sql"
    echo -n "Do you wish to run this against ${NAME2}? [yN]: "; read x
    if [[ $x =~ y ]]; then
       pgrun -U $USER2 -pw $PASS2 -h $HOST2 -d $NAME2 -o "$OPT2" -f "${TYPE}.sql"
    fi
}

#rundiff ROLE
#rundiff SEQUENCE
#rundiff TABLE
#rundiff COLUMN
#rundiff INDEX
#rundiff FOREIGN_KEY
#rundiff GRANT
rundiff OWNER

