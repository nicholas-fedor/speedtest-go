Name:           speedtest-go
Version:        %{version}
Release:        %{release}%{?dist}
Summary:        Command-line internet speed test tool
License:        MIT
BuildArch:      x86_64

%description
speedtest-go is a command-line internet speed test tool written in Go,
using the Speedtest.net infrastructure.

%install
install -D -m 755 %{_builddir}/speedtest-go %{buildroot}/usr/bin/speedtest-go

%files
/usr/bin/speedtest-go

%changelog
* %(date "+%a %b %d %Y") Build System <build@localhost> - %{version}-%{release}
- Automated package build
