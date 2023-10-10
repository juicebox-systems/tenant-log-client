# Tenant Log Client

This is a test client for the tenant log API exposed by the realms.

Usage of ./tenant-event-log:
  -ack
    	send ack for received events
  -page int
    	page size (default 1)
  -token string
    	Auth token
  -url string
    	URL to the tenant event service (default "http://localhost:8080")
  -watch
    	continue to poll and watch for events

If `-watch` is specified, then the client will continue to poll for events
otherwise only a single request is made.

If `-ack` is set, then any events received will be ack'd (and therefore deleted
on the server).

Using `-watch` without `-ack` will work, but can seem odd as the unack'd events
will continue to re-appear when the ack window (10 seconds) times out.
