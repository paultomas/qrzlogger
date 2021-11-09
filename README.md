# qrzlogger

This is a little logging utility that works in conjunction with wsjtx. You start it up whenever you run wsjtx. This utility will then listen for events from wsjtx. When it picks up a "logged QSO" event, it will upload that QSO to your QRZ logbook.

If you work offline, it will fail to upload to QRZ, and fallback to storing the logbook entries into a database file so that when you start it up again, this time connected to the network, it will send those entries over to your QRZ logbook.
 

Running qrzlogger
----------------

    Usage of ./qrzlogger:
    -d string
    	Database file (default ".qrzlogger.sqlite3")
    -h string
    	host ip (default "0.0.0.0")
    -k string
    	API key
    -p int
    	port (default 2237)
