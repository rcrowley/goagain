set -e -x

findproc() {
    set +x
    find "/proc" -mindepth 2 -maxdepth 2 -name "exe" -lname "$PWD/inherit" 2>"/dev/null" |
    cut -d"/" -f"3"
    set -x
}

cd "example/inherit"
go build
./inherit &
PID="$!"
[ "$PID" -a -d "/proc/$PID" ]
for _ in _ _
do
    OLDPID="$PID"
    sleep 1
    kill -USR2 "$PID"
    sleep 2
    PID="$(findproc "inherit")"
    [ ! -d "/proc/$OLDPID" -a "$PID" -a -d "/proc/$PID" ]
done
[ "$(nc "127.0.0.1" "48879")" = "Hello, world!" ]
kill -TERM "$PID"
sleep 2
[ ! -d "/proc/$PID" ]
[ -z "$(findproc "inherit")" ]
cd "$OLDPWD"
