test-uniqueid: 
	go build -o bin/unique-id cmd/unique-id/main.go && bin/maelstrom/maelstrom test -w unique-ids --bin bin/unique-id --time-limit 30 --rate 100000 --node-count 100 --availability total --nemesis partition

test-g-counter: 
	go build -o bin/g-counter cmd/g-counter/main.go && bin/maelstrom/maelstrom test -w g-counter --bin bin/g-counter --time-limit 20 --rate 1000 --node-count 100 --nemesis partition

test-broadcast: 
	go build -o bin/broadcast cmd/broadcast/main.go && bin/maelstrom/maelstrom test -w broadcast --bin bin/broadcast --time-limit 20 --rate 10 --node-count 5 

install: 
	curl -L https://github.com/jepsen-io/maelstrom/releases/download/v0.2.3/maelstrom.tar.bz2 --create-dirs --output bin/maelstrom.tar.bz2 && tar -xvf bin/maelstrom.tar.bz2 -C bin && rm bin/maelstrom.tar.bz2