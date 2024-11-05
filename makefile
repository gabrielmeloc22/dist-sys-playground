test-uniqueid: 
	go build -o bin/unique-id cmd/unique-id/main.go && bin/maelstrom/maelstrom test -w unique-ids --bin bin/unique-id --time-limit 30 --rate 10000 --node-count 100 --availability total --nemesis partition

install: 
	curl -L https://github.com/jepsen-io/maelstrom/releases/download/v0.2.3/maelstrom.tar.bz2 --create-dirs --output bin/maelstrom.tar.bz2 && tar -xvf bin/maelstrom.tar.bz2 -C bin && rm bin/maelstrom.tar.bz2