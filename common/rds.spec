################################################################################

%define debug_package  %{nil}

################################################################################

%define redis_user  redis

################################################################################

Summary:        Redis orchestration tool
Name:           rds
Version:        1.2.0
Release:        0%{?dist}
Group:          Applications/System
License:        Apache License, Version 2.0
Vendor:         ESSENTIAL KAOS
URL:            https://kaos.sh/rds

Source0:        https://source.kaos.st/%{name}/%{name}-%{version}.tar.bz2

BuildRoot:      %{_tmppath}/%{name}-%{version}-%{release}-root-%(%{__id_u} -n)

BuildRequires:  golang >= 1.20

Requires:       tuned

Provides:       %{name} = %{version}-%{release}

################################################################################

%description
Tool for Redis orchestration.

################################################################################

%package sync
Summary:   RDS Sync daemon
Version:   1.1.1
Release:   0%{?dist}
Group:     Applications/System

Requires:  %{name}

Provides:  %{name}-sync = %{version}-%{release}

%description sync
RDS Sync daemon.

################################################################################

%prep
%setup -q

%build
if [[ ! -d "%{name}/vendor" ]] ; then
  echo "This package requires vendored dependencies"
  exit 1
fi

pushd %{name}
  %{__make} %{?_smp_mflags} all
popd

%install
rm -rf %{buildroot}

install -dm 755 %{buildroot}%{_bindir}
install -dm 755 %{buildroot}%{_localstatedir}/log
install -dm 755 %{buildroot}%{_mandir}/man1

install -dm 755 %{buildroot}/opt/%{name}/conf
install -dm 755 %{buildroot}/opt/%{name}/data
install -dm 755 %{buildroot}/opt/%{name}/log
install -dm 755 %{buildroot}/opt/%{name}/meta
install -dm 755 %{buildroot}/opt/%{name}/pid
install -dm 755 %{buildroot}/opt/%{name}/templates

pushd %{name}
  install -pDm 640 common/%{name}.knf \
                   %{buildroot}%{_sysconfdir}/%{name}.knf
  install -pDm 644 common/%{name}.logrotate \
                   %{buildroot}%{_sysconfdir}/logrotate.d/%{name}

  install -pm 755 %{name} %{buildroot}%{_bindir}/
  install -pm 755 %{name}-sync %{buildroot}%{_bindir}/

  install -pDm 644 common/%{name}-sync.service \
                   %{buildroot}%{_unitdir}/%{name}-sync.service

  install -pDm 644 common/tuning/tuned.conf \
                   %{buildroot}%{_sysconfdir}/tuned/no-thp/tuned.conf
  install -pDm 755 common/tuning/no-defrag.sh \
                   %{buildroot}%{_sysconfdir}/tuned/no-thp/no-defrag.sh
  install -pDm 644 common/tuning/50-rds.sysctl \
                   %{buildroot}%{_sysconfdir}/sysctl.d/50-rds.conf
  install -pDm 644 common/tuning/50-rds.limits \
                   %{buildroot}%{_sysconfdir}/security/limits.d/50-rds.conf

  cp -r common/templates/* %{buildroot}/opt/%{name}/templates/

  ./%{name} --generate-man > %{buildroot}%{_mandir}/man1/%{name}.1
  ./%{name}-sync --generate-man > %{buildroot}%{_mandir}/man1/%{name}-sync.1
popd

ln -s /opt/%{name}/log \
      %{buildroot}%{_localstatedir}/log/%{name}

%clean
rm -rf %{buildroot}

%pre
getent group redis &> /dev/null || groupadd -r redis &> /dev/null
getent passwd redis &> /dev/null || \
useradd -r -g redis -d %{_sharedstatedir}/redis -s /sbin/nologin \
        -c 'Redis Server' redis &> /dev/null

%post
if [[ -d %{_sysconfdir}/bash_completion.d ]] ; then
  %{name} --completion=bash 1> %{_sysconfdir}/bash_completion.d/%{name} 2>/dev/null
fi

if [[ -d %{_datarootdir}/fish/vendor_completions.d ]] ; then
  %{name} --completion=fish 1> %{_datarootdir}/fish/vendor_completions.d/%{name}.fish 2>/dev/null
fi

if [[ -d %{_datadir}/zsh/site-functions ]] ; then
  %{name} --completion=zsh 1> %{_datadir}/zsh/site-functions/_%{name} 2>/dev/null
fi

%postun
if [[ $1 == 0 ]] ; then
  if [[ -f %{_sysconfdir}/bash_completion.d/%{name} ]] ; then
    rm -f %{_sysconfdir}/bash_completion.d/%{name} &>/dev/null || :
  fi

  if [[ -f %{_datarootdir}/fish/vendor_completions.d/%{name}.fish ]] ; then
    rm -f %{_datarootdir}/fish/vendor_completions.d/%{name}.fish &>/dev/null || :
  fi

  if [[ -f %{_datadir}/zsh/site-functions/_%{name} ]] ; then
    rm -f %{_datadir}/zsh/site-functions/_%{name} &>/dev/null || :
  fi
fi

%preun sync
if [[ $1 -eq 0 ]] ; then
  systemctl stop %{name}-sync.service &>/dev/null || :
fi

%postun sync
systemctl daemon-reload &>/dev/null || :

################################################################################

%files
%defattr(-, root, root, -)
%doc %{name}/LICENSE
%dir /opt/%{name}/meta
%dir /opt/%{name}/conf
%dir /opt/%{name}/data
%dir /opt/%{name}/log
%attr(-, %{redis_user}, %{redis_user}) /opt/%{name}/pid
%config(noreplace) %{_sysconfdir}/tuned/no-thp/tuned.conf
%config(noreplace) %{_sysconfdir}/sysctl.d/50-rds.conf
%config(noreplace) %{_sysconfdir}/security/limits.d/50-rds.conf
%config(noreplace) %{_sysconfdir}/%{name}.knf
%config(noreplace) %{_sysconfdir}/logrotate.d/%{name}
%config(noreplace) %{_localstatedir}/log/%{name}
%config(noreplace) /opt/%{name}/templates/redis/*.conf
%config(noreplace) /opt/%{name}/templates/sentinel/*.conf
%{_sysconfdir}/tuned/no-thp/no-defrag.sh
%{_mandir}/man1/%{name}.1.*
%{_bindir}/%{name}

%files sync
%defattr(-, root, root, -)
%{_mandir}/man1/%{name}-sync.1.*
%{_unitdir}/%{name}-sync.service
%{_bindir}/%{name}-sync

################################################################################

%changelog
* Sun Sep 10 2023 Anton Novojilov <andy@essentialkaos.com> - 1.2.0-0
- [cli] Added 'validate-templates' command for templates validation
- [cli] Added 'backup-create' command for creating RDB snapshots
- [cli] Added 'backup-restore' command for restoring instance data from
  snapshots
- [cli] Added 'backup-clean' command for deleting RDB snapshots
- [cli] Added 'backup-list' command for listing RDB snapshots
- [cli] Added -R/--raw option for forcing raw output
- [sync] Disable read-only mode for replicas on minion if standby failover
  is used
- [core] Run all processes with umask 027
- [core] Use sync.Map for caching metadata
- [cli] Improved properties filtering in 'conf' command
- [cli] Added using of password variations for password auth
- [cli] Improved actions logging
- [cli] Fixed raw output from 'top' command for long numbers
- [cli] Fixed command execution with 'cli'
- [cli] Fixed password check using password variations
- [cli] Fix password check using password variations

* Tue Aug 22 2023 Anton Novojilov <andy@essentialkaos.com> - 1.1.0-0
- Added info about RDS to templates payload
- Added instance storage data to templates payload

* Mon Aug 21 2023 Anton Novojilov <andy@essentialkaos.com> - 1.0.0-0
- First public release
