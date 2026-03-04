#!/bin/bash

# Rename old tables to _old suffix
# Then copy data to new GORM tables

HOST="172.17.0.1"
PORT="3367"
USER="hajo4kids"
PASS='[_wEahKPxkI[2h1P'
DB="hajo4kids"

echo "=== Renaming old tables ==="

# Rename tables
mysql -h $HOST -P $PORT -u $USER -p"$PASS" $DB << 'EOF'
RENAME TABLE users TO users_old;
RENAME TABLE kategorien TO kategorien_old;
RENAME TABLE ziele TO ziele_old;
RENAME TABLE bilder TO bilder_old;
RENAME TABLE marketers TO marketers_old;
RENAME TABLE events TO events_old;
RENAME TABLE trip TO trip_old;
RENAME TABLE favoriten TO favoriten_old;
RENAME TABLE rating TO rating_old;
RENAME TABLE ziele_kategorien TO ziele_kategorien_old;
RENAME TABLE ziele_bilder TO ziele_bilder_old;
RENAME TABLE ziele_trip TO ziele_trip_old;
EOF

echo "=== Tables renamed ==="
echo ""
echo "Now run the backend to create new tables via GORM AutoMigrate"
echo "Then run: ./copy-tables -host $HOST -port $PORT -user $USER -pass '$PASS' -db $DB"