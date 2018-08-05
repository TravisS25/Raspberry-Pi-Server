#/usr/bin/python3.5
import time
import requests
from datetime import datetime
import os
from shutil import copyfile
import sqlite3
import sys
import shutil
import device
import getopt
import configparser
import random


CONFIG = configparser.ConfigParser()
PROJECT_NAME = ".raspberry_pi_client"
project_root = os.path.join(os.path.expanduser('~'), PROJECT_NAME)
client_config_file = os.path.join(project_root, "client.ini")
csv_directory = os.path.join(project_root, "csv")
sets_directory = os.path.join(csv_directory, "sets")


def _check_in_device(pi_device):
    """
    Takes an instance of Device and CONFIG and tries to 
    check in current running device to the server so it can begin
    sending local information to the server 

    This function is called when starting the device and if it fails 
    to check in on start up due to not being connected to the internet,
    the server is down etc. then this function will continually be called 
    in the main for loop until it is checked in.  It will continue to write
    to local csv whether it checks in or not
    """

    payload = {"password": pi_device.password, "deviceName": pi_device.device_name}
    check_in_url = pi_device.protocol + pi_device.ip_address + pi_device.port + "/check-in-handler/"
    print("checkin url " + check_in_url)
    try:
        print("sending request to check in")
        r = requests.post(check_in_url, data=payload)
        result = r._content.decode('UTF-8')

        # If server responds that device is already checked in, we will continue
        # to write locally 
        # Else we tell current device it is signed in
        if result == "Already checked in":
            CONFIG["device"]["is_checked_in"] = "False"
            print("Device name is already in use, not sending to server but still running locally...")
        else:
            pi_device.is_checked_in = True
            CONFIG["device"]["is_checked_in"] = "True"
    
    # Reaches exception if we could not connect to server
    except Exception as e:
        print(e)
        print("Have no internet trying to checking in but still going...")
        pi_device.has_internet = False
        pi_device.had_internet_before = False
        CONFIG["device"]["has_internet"] = "False"
        CONFIG["device"]["had_internet_before"] = "False"

    with open(client_config_file, "w+") as config_file:
        CONFIG.write(config_file)
    

def _get_pi_device():
    """
    Initialize a pi device instance with info from the CONFIG file
    Will print error message if any of the CONFIG settings are not filled out
    or not correct and will exit the program
    """
    
    check_file_message = "Check client.ini file or documentation for more"
    error = False

    try:
        device_name = CONFIG["DEFAULT"]["device_name"]
        ip_address = CONFIG["DEFAULT"]["ip_address"]
        port = CONFIG["DEFAULT"]["port"]
        sleep = CONFIG["DEFAULT"]["sleep"]
        https = CONFIG["DEFAULT"]["https"]
        password = CONFIG["DEFAULT"]["password"]
        has_internet = CONFIG["device"]["has_internet"]
        had_internet_before = CONFIG["device"]["had_internet_before"]
        has_new_set_not_recording = CONFIG["device"]["has_new_set_not_recording"]
        is_recording = CONFIG["device"]["is_recording"]
        device_set = CONFIG["device"]["device_set"]
    except KeyError as ex:
        print(ex)
        print("Stopping Device...")
        sys.exit(2)

    # Go through each CONFIG parameter and determine if it's correct 
    if not device_name:
        print("device_name needs to be given.  " + check_file_message)
        error = True
    if not ip_address:
        print("ip_address needs to be given.  " + check_file_message)
        error = True
    if not port:
        print("port needs to be given.  " + check_file_message)
        error = True
    if not sleep:
        print("sleep needs to be given.  " + check_file_message)
        error = True
    try:
        sleep = float(sleep)
    except ValueError:
        print("sleep must be an float.  " + check_file_message)
        error = True
    if float(sleep) < 0:
        print("sleep must be greater than 0.  " + check_file_message)
        error = True
    if not has_internet:
        print("has_internet settings is needed.  " + check_file_message)
        error = True
    try: 
        has_internet = bool(has_internet)
    except ValueError as e:
        print("has_internet should be boolean.  " + check_file_message)
        error = True
    if not had_internet_before:
        print("had_internet_before settings is needed.  " + check_file_message)
        error = True
    try:
        had_internet_before = bool(had_internet_before)
    except ValueError as e:
        print("had_internet_before should be boolean.  " + check_file_message)
        error = True
    if not has_new_set_not_recording:
        print("has_new_set_not_recording setting is needed.  " + check_file_message)
        error = True
    try:
        has_new_set_not_recording = bool(has_new_set_not_recording)
    except ValueError as e:
        print("has_new_set_not_recording should be boolean.  " + check_file_message)
        error = True
    if not is_recording:
        print("is_recording setting is needed.  " + check_file_message)
        error = True
    try:
        is_recording = bool(is_recording)
    except ValueError as e:
        print("is_recording should be boolean.  " + check_file_message)
        error = True
    if not device_set:
        print("device_set setting is needed.  " + check_file_message)
        error = True
    try:
        device_set = int(device_set)

        if device_set < 1:
            print("device_set should be greater than 0.  " + check_file_message)
            error = True
    except ValueError as e:
        print("device_set should be int.  " + check_file_message)
        error = True
    if not password:
        print("password should be set.  " + check_file_message)
        error = True
    if len(password) < 5:
        print("password has to be at least 6 characters long.  " + check_file_message)
        error = True
    try:
        https = https.lower()

        if https == "true":
            https = True
        elif https == "false":
            https = False
        else:
            raise ValueError
    except ValueError as e:
        print("https should be boolean.  " + check_file_message)
        error = True

    if error:
        print("Stopping device...")
        sys.exit(2)

    print("https " + str(https))

    pi_device = device.Device(
        device_name=device_name,
        ip_address=ip_address,
        sleep=sleep, 
        is_recording=is_recording,
        has_internet=has_internet,
        had_internet_before=had_internet_before,
        has_new_set_not_recording=has_new_set_not_recording,
        current_set=device_set,
        password=password,
        https=https
    )
    return pi_device


def _init_file_system():
    """
    Initializes file system with proper directories and files needed
    """
    print("This is the first time you are running this program.  We will now set some defaults.")
    device_name = input("Enter device name.  NOTE - this must be unique if connecting multiple "
    "pi devices (default Device):")
    ip_address = input("Enter ip address of server pic device will connect to (default localhost):")
    port = input("Enter port number the server is binded to (default :8003):")
    password = input("Enter password that is used for server (default password):")

    if not device_name:
        device_name = "Device"
    if not password:
        password = "password"
    if not ip_address:
        ip_address = "localhost"
    if not port:
        port = ":8003"

    os.makedirs(sets_directory)
    config_file = open(client_config_file, "w+")
    csv_file = os.path.join(csv_directory, device_name + ".csv")
    with open(csv_file, "w+") as f:
        pass

    write_to_file = \
    """
    [DEFAULT]
    device_name = """ + device_name + """
    password = """ + password + """
    ip_address = """ + ip_address + """
    port = """ + port + """
    sleep = 2
    https = False

    [device]
    has_internet = True
    had_internet_before = True
    has_new_set_not_recording = False
    is_recording = True
    device_set = 1
    is_checked_in = True
    """

    config_file.write(write_to_file)
    config_file.close()


def init(pi_device):
    """
    Start of the program
    """
    while True:
        # Sleep sets the interval in which the sensors try to detect for movement
        time.sleep(pi_device.sleep)
        payload = {"password": pi_device.password}
        print("init recording " + str(pi_device.is_recording))

        # If device is recording, we first check if it was recording before
        # and if it wasn't then this indicates that the device wasn't communicating 
        # with the server before.  Therefore we resend the whole csv file to the server
        # to re-update the server
        if pi_device.is_recording:
            print("device set from file - recording " + CONFIG["device"]["device_set"])
            print("device set from object - recording " + str(pi_device.current_set))
            pi_device.has_new_set_not_recording = False
            CONFIG["device"]["is_recording"] = "True"
            CONFIG["device"]["has_new_set_not_recording"] = "False"

            if pi_device.has_internet and not pi_device.had_internet_before:
                print("reloading csv file")
                with open(pi_device.csv_file, 'rb') as f:
                    payload.update({"fileName": pi_device.device_name + ".csv"})
                    reload_url = pi_device.protocol + pi_device.ip_address + pi_device.port + "/reload-csv/"
                    print(str(payload))
                    r = requests.post(
                        reload_url, 
                        payload,
                        files={"uploadFile": f}
                    )

                pi_device.had_internet_before = True
                CONFIG["device"]["had_internet_before"] = "True"

            # Movement will be replaced by motion sensor code that indicates
            # if there had been movement or not
            movement = random.getrandbits(1)
            print("movement " + str(movement))
            current_date = datetime.now().strftime("%Y-%m-%d")
            current_time = datetime.now().strftime("%H:%M:%S")
            time_stamp = pi_device.device_name + "," + \
                            current_date + "," + \
                            current_time + "," + \
                            str(movement) 

            if movement == 1:
                with open(pi_device.csv_file, 'a') as f:
                    time_stamp_array = time_stamp.split(",")
                    new_time_stamp = time_stamp_array[1] + "," + time_stamp_array[2] + " \n"
                    f.write(new_time_stamp)
                    print("Writing to file...")

            sensor_url = pi_device.protocol + pi_device.ip_address + pi_device.port + "/sensor-handler/"
            payload.update({"timeStamp": time_stamp})
        
            try:
                # If device is considered checked in, send info to server and get response
                # Possible responses we can receive are:
                #   - Stop Recording: Indicates that current device should stop recording altogether
                #   - New Set: Indicates that current device should start a new set which stores current
                #       csv file in directory and erases current csv contents
                #   - Wrong Password: Indicates that current password set in CONFIG file is incorrect
                #   - Device does not exist: This indicates that the current device did not try to 
                #       check in to the server  
                if pi_device.is_checked_in:
                    r = requests.post(sensor_url, payload)
                    print("Sending to server...")
                    response = str(r._content.decode("utf-8")).split(",")
                    print("response " + str(response))

                    for item in response:
                        if item == "Stop Recording":
                            pi_device.is_recording = False
                            CONFIG["device"]["is_recording"] = "False"
                        if item == "Wrong Password":
                            print(item + ", not writing to server but still locally...")
                        if item == "Device does not exist":
                            print(item + " on server.  Please check in device.  Still writing locally...")

                    pi_device.has_internet = True
                    CONFIG["device"]["has_internet"] = "True"
                # Else device will try to check in to server
                else:
                    _check_in_device(pi_device)
            
            # Will reach exception if could not connect to server
            except Exception as e:
                print("Have no internet within recording but still going...")
                pi_device.has_internet = False
                pi_device.had_internet_before = False
                CONFIG["device"]["has_internet"] = "False"
                CONFIG["device"]["had_internet_before"] = "False"

            with open(client_config_file, "w+") as config_file:
                CONFIG.write(config_file)

        # Else device is not recording so we ping to see if status is updated
        # for device to start recording again
        else:
            CONFIG["device"]["is_recording"] = "False"
            print("device set from file - not recording " + CONFIG["device"]["device_set"])
            print("device set from object - not recording " + str(pi_device.current_set))

            try:
                r = requests.get(
                    pi_device.protocol + pi_device.ip_address + pi_device.port + "/device-status-handler/",
                    params={"deviceName": pi_device.device_name}
                )
                print("Not recording but still going...")
                response = str(r._content.decode("utf-8")).split(",")

                # Response we will receive from server are:
                #   - Record: Indicates that the device should start recording again
                #   - New Set: Indicates that the device should start a new csv set
                for item in response:
                    if item == "Record":
                        pi_device.is_recording = True
                        print("Some how getting here")
                        CONFIG["device"]["is_recording"] = "True"
                    if item == "New Set" and not pi_device.has_new_set_not_recording:
                        print("new set while not recording")
                        set_csv_file = os.path.join(sets_directory, str(pi_device.current_set) + ".csv")
                        
                        # Simply creating new set file that we will copy current csv contents to
                        with open(set_csv_file, "w+"):
                            pass

                        # print("previous current set " + str(pi_device.current_set))
                        pi_device.current_set = pi_device.current_set + 1
                        # print("new current set " + str(pi_device.current_set))
                        CONFIG["device"]["device_set"] = str(pi_device.current_set)
                        pi_device.has_new_set_not_recording = True
                        CONFIG["device"]["has_new_set_not_recording"] = "True"
                    
                        copyfile(pi_device.csv_file, set_csv_file)
                        csv_file = open(pi_device.csv_file, 'w')
                        csv_file.close()

            # Reach exception if could not connect to server
            except Exception as e:
                print(e)
                pi_device.has_internet = False
                pi_device.had_internet_before = False
                CONFIG["device"]["has_internet"]
                CONFIG["device"]["had_internet_before"]
                print("Not recording and no internet but still going...")

            with open(client_config_file, "w+") as config_file:
                CONFIG.write(config_file)


def read_config_file():
    # client.ini file has to exist or program will exist
    if not os.path.exists(client_config_file):
        print("Must have client.ini file in project root directory")
        print("Stopping device...")
        sys.exit(2)

    CONFIG.read(client_config_file)


if __name__ == '__main__':
    if not os.path.exists(project_root):
        _init_file_system()
    read_config_file()
    pi_device = _get_pi_device()

    # Get a list of command line arguments and determine what to 
    # do with them 
    try:
        opts, args = getopt.getopt(sys.argv[1:],"w",["wipe"])
    except getopt.GetoptError:
        print('client.py -w --wipe')
        sys.exit(2)

    for opt, arg in opts:
        # If -w or --wipe is passed, we will receive message indicating whether 
        # we want to continue to delete all csv files
        # This option should only be used for testing for a simple way to delete
        # all csv content without manually having to do it and SHOULD NOT be used
        # in production
        if opt in ("-w", "--wipe"):
            answer = input("You are about to delete current and all csv files.  Are you sure you want to continue? (y/n)")
            if answer == "y" or answer == "Y":
                print("Client wiped")
                if os.path.exists(project_root):
                    # Extra saftey to make sure we don't delete home directory
                    if project_root != os.path.expanduser('~'):
                        shutil.rmtree(project_root)
                    else:
                        print("Can't delete home directory, thats dangerous!")
            else:
                print("Nothing was deleted")

    _check_in_device(pi_device)
    init(pi_device)


                