# Fileserver

Fileserver serves your files over your private network. It is great for sharing resources over your network, including your workstations, mobile devices, and e-readers like Kobo, removing the need to upload to an intermediate server like Google Drive or Dropbox. Fileserver is written in Go and it uses NO EXTERNAL LIBRARIES.

## Installation

It is recommended to build by running the go build command on your system (assuming you have Go installed):

Clone this repository, and in the folder run this:

```bash
go build .
```

## Quick Download

Some builds are available for download, click on your architecture to get started:

**Open Link and Click on Raw**

[Windows, AMD64](https://github.com/joshuaetim/fileserver/blob/main/builds/fileserver_win_amd64.exe)

[Windows, ARM](https://github.com/joshuaetim/fileserver/blob/main/builds/fileserver_win_arm.exe)

[Darwin, ARM](https://github.com/joshuaetim/fileserver/blob/main/builds/fileserver_mac)

(_Please reach out if you want your platform build included_)

---

## Usage

Double-click on the executable and a terminal should open with the prompt message (different based on your user account):

```bash
enter path of book: (press enter for default home)  /Users/
```
Press enter for the default HOME directory, or enter a specific directory. After that, a welcome message should appear:

```bash
__        __   _                                    
\ \      / /__| | ___ ___  _ __ ___   ___    
 \ \ /\ / / _ \ |/ __/ _ \| '_ ' _ \ / _ \    
  \ V  V /  __/ | (_| (_) | | | | | |  __/
   \_/\_/ \___|_|\___\___/|_| |_| |_|\___|


server started on  0.0.0.0:3000
Enter http://172.20.10.4:3000 in your browser
```

Enter the URL on any device and browse your files.

## Contributing

Pull requests are welcome. For major changes, please open an issue first
to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License

[MIT](https://choosealicense.com/licenses/mit/)
