
all:

build:
	( cd listen ; go build )
	( cd publish ; go build )
	( cd tracer-bot ; go build )

