cd web/frontend
bun install
bun run build

cd ../..
go build -ldflags="-s -w" -o pt-tools.exe .
