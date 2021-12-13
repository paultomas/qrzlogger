# qrzlogger

This is a little logging utility that works in conjunction with wsjtx. You start it up whenever you run wsjtx. This utility will then listen for events from wsjtx, by default on port 2237. When it picks up a "logged QSO" event, that is, after you have clicked "OK" in wsjtx's "log QSO" dialog,  it will upload that QSO to your QRZ logbook.

If you work offline, it will fail to upload to QRZ, and fallback to storing the logbook entries into a database file so that when you start it up again, this time connected to the network, it will send those entries over to your QRZ logbook.
 
Building qrzlogger
----------------

To build qrzlogger you need Golang installed. You can visit https://golang.org/dl/ to download Golang.

1. clone this repo
2. `cd qrzlogger`
3. `go build`

The binary called 'qrzlogger' will be placed in the root folder of this project.



Running qrzlogger
----------------

    Usage of ./qrzlogger:
    -d string
    	Database file (default "~/.qrzlogger.sqlite3")
    -h string
    	host ip (default "0.0.0.0")
    -p int
    	port (default 2237)

You need to have a QRZ subscription and an API key.

Set the environment variable QRZ_KEY to your API key.

E.g. a typical invocation (Linux/macOS):
    export QRZ_KEY=<your-API-key>
    ./qrzlogger

Windows:
    set QRZ_KEY=<your-API-key>
    qrzlogger
    

You can use:
    
    ./qrzlogger --help

to see the above usage information.
