graphite_perf_test_go
==================

Simple load testing utility for graphite, Go-based

Modified to be multithreaded by me.
Thanks for some optimizations and help in creating initial release to -=Serj=- (https://github.com/sshikaree/)

Example
==================
./graphite_perf_test_go -connections=110000 -host=[::1]:2024 -simul=1000 -points=1000 -runs=2

Will create load of 110M metrics/minute (datapoints/minute), 110k TCP connections (1000 simultaniously) and 1000 datapoints/connection on IPv6 addr ::1 (localhost). It will also run 2 times and print mean value and stddev.

Performacne
==================
2 Cores of Ivy Brdige-E (Xeon E5-2650v2 2.6GHz) can create 110M metrics per minute and bottleneck is 1GBPS network card.
