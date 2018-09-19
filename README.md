This is an app that monitors callerid info and ring status and sends them to an MQTT server
for processing by a home automation controller

To make it start as a service, copy callerid.service to /etc/systemd/system, then edit that script to point to the callerid executable.

Then run the following:

$ sudo systemctl daemon-reload
$ sudo systemctl enable callerid.service
