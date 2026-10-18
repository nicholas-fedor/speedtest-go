# A full speed test, with its live progress and the result panel. --mask-isp hides the IP address, ISP name, and
# coordinates. Server 49365 in San Jose answers the packet loss protocol, so every result is measured.
Output docs/static/images/demo.webm
Output docs/static/images/demo.mp4
Output docs/static/images/demo.gif

Source docs/tapes/settings.tape
Set Height 560

# speedtest-go comes from bin/, which task docs:demo builds with the latest release's version.
Hide
Type "export PATH=$PWD/bin:$PATH && clear"
Enter
Show

Sleep 500ms
Type "speedtest-go --server 49365 --mask-isp"
Sleep 500ms
Enter

# The test is done when the shell prompt returns.
Wait@120s
Sleep 4s
