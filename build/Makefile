generate_proto:
	@../tools/protoc --plugin=protoc-gen-go=../tools/protoc-gen-go --go_out=../ -I../pb ../pb/raft.proto
	@../tools/protoc --plugin=protoc-gen-raftext=../tools/protoc-gen-raftext --raftext_out=../ -I../pb ../pb/raft.proto
	