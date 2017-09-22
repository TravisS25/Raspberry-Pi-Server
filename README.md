# Raspberry Pi Motion Sensor Server
### Dependencies
1.  sqlite3
2.  python3.5
3.  Go 1.8

This is a small project that I did for a friend for their biology thesis.  Server side is written in go and the client side in python.  
This project allows a user to set up a server and connect an arbitray amount of pi devices that have motion sensors attached to them and writes to csv file the times in which movement is detected.  The information is also written locally to a .csv, and sent to server in the format:

<string|device_name>, <string|date>, <string|time>, <int|bool|motion_detected>

The client will send this information based on -s --sleep option passed (.5 default) whether motion is detected or not.   

On the server, it takes the information passed and writes it to its own csv file based on device name but again only prints when motion is detected.  Reason for constant pinging no matter if motion is detected or not is to simply indicate that the device is still on/running.  A timestamp is kept for each device and has a default timeout of 5 sec so if a device is not heard from after that, a warning message with the device name and timestamp of the last time its been heard from will pop up on the webpage.

 