git reset --hard
git pull
go build -o ./build
chmod +x ./build/am-clanactivity
./build/am-clanactivity
