################################################################################

%define debug_package  %{nil}

################################################################################

%define redis_user  redis

################################################################################

Summary:        Redis orchestration tool
Name:           rds
Version:        1.11.2
Release:        0%{?dist}
Group:          Applications/System
License:        Apache License, Version 2.0
Vendor:         ESSENTIAL KAOS
URL:            https://kaos.sh/rds

Source0:        https://source.kaos.st/%{name}/%{name}-%{version}.tar.bz2

BuildRoot:      %{_tmppath}/%{name}-%{version}-%{release}-root-%(%{__id_u} -n)

BuildRequires:  golang >= 1.22

Requires:       tuned

Provides:       %{name} = %{version}-%{release}

################################################################################

%description
Tool for Redis orchestration.

################################################################################

%package sync
Summary:   RDS Sync daemon
Version:   1.4.2
Release:   0%{?dist}
Group:     Applications/System

Requires:  %{name}

Provides:  %{name}-sync = %{version}-%{release}

%description sync
RDS Sync daemon.

################################################################################

%prep

%setup -q
if [[ ! -d "%{name}/vendor" ]] ; then
  echo -e "----\nThis package requires vendored dependencies\n----"
  exit 1
elif [[ -f "%{name}/%{name}" || -f "%{name}/%{name}-sync" ]] ; then
  echo -e "----\nSources must not contain precompiled binaries\n----"
  exit 1
fi

%build
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
  install -pDm 640 common/tuning/50-rds.sudoers \
                   %{buildroot}%{_sysconfdir}/sudoers.d/50-rds

  cp -r common/templates/* %{buildroot}/opt/%{name}/templates/

  ./%{name} --generate-man > %{buildroot}%{_mandir}/man1/%{name}.1
  ./%{name}-sync --generate-man > %{buildroot}%{_mandir}/man1/%{name}-sync.1
popd

ln -s /opt/%{name}/log \
      %{buildroot}%{_localstatedir}/log/%{name}

%clean
rm -rf %{buildroot}

%pre
getent group %{name} &> /dev/null || groupadd -r %{name} &> /dev/null
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
%config(noreplace) %{_sysconfdir}/sudoers.d/50-rds
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
%config(noreplace) %{_unitdir}/%{name}-sync.service
%{_mandir}/man1/%{name}-sync.1.*
%{_bindir}/%{name}-sync

################################################################################

%changelog
* Fri Nov 01 2024 Anton Novojilov <andy@essentialkaos.com> - 1.11.2-0
- [cli|sync] Fixed bug with daemonizing Redis server
- Dependencies update
- Code refactoring

* Thu Sep 26 2024 Anton Novojilov <andy@essentialkaos.com> - 1.11.1-0
- Collect more data for verbose version info
- Dependencies update
- Code refactoring

* Thu Jul 11 2024 Anton Novojilov <andy@essentialkaos.com> - 1.11.0-0
- [cli] Added 'uptime' command
- [cli] Improved output of 'stats-command' command
- [cli] Improved output of 'help' command
- Package ek updated to v13
- Code refactoring

* Wed May 08 2024 Anton Novojilov <andy@essentialkaos.com> - 1.10.5-0
- Dependencies update

* Sat Apr 27 2024 Anton Novojilov <andy@essentialkaos.com> - 1.10.4-0
- Fixed output of sections with no properties in 'settings' command
- Fixed output of empty value in 'settings' command for keepalived
- Code refactoring
- Dependencies update

* Sat Mar 30 2024 Anton Novojilov <andy@essentialkaos.com> - 1.10.3-0
- [sync] Improved systemd support
- Code refactoring
- Dependencies update

* Fri Mar 22 2024 Anton Novojilov <andy@essentialkaos.com> - 1.10.2-0
- [cli] Fixed bug with checking keepalived status while changing node role
- [sync] Fixed bug with updating instances states on minion
- [cli] [sync] Improved support information gathering
- Dependencies update

* Fri Jan 26 2024 Anton Novojilov <andy@essentialkaos.com> - 1.10.1-0
- [cli] Fixed bug with printing separator in 'log' output
- [cli] Allow to run 'go' command on minions

* Tue Jan 23 2024 Anton Novojilov <andy@essentialkaos.com> - 1.10.0-0
- [cli] Fixed bug with showing current Redis version in 'info' command output
- [cli] [sync] Send info about command initiator
- [cli] Improved 'stats' command output
- [cli] Added pager usage configuration to user preferences

* Mon Jan 22 2024 Anton Novojilov <andy@essentialkaos.com> - 1.9.0-0
- [cli] Improved settings info rendering
- [cli] UI improvements
- Dependencies update

* Fri Jan 19 2024 Anton Novojilov <andy@essentialkaos.com> - 1.8.2-0
- Dependencies update

* Fri Jan 19 2024 Anton Novojilov <andy@essentialkaos.com> - 1.8.1-0
- [cli] Fixed bug with using raw data output when output device is not TTY
- [cli] [sync] Minor UI improvements
- Dependencies update

* Wed Dec 13 2023 Anton Novojilov <andy@essentialkaos.com> - 1.8.0-0
- [cli] [sync] Added integration with keepalived
- [sync] Better command processing logging
- [core] Fixed bug with changing owner of non-existent instance log file

* Sat Dec 02 2023 Anton Novojilov <andy@essentialkaos.com> - 1.7.0-0
- [cli] Added '-P'/'--pager' option to use pager for long output
- [cli] Improved 'help' command output
- [cli] Fixed bug with separator in 'list' command output

* Fri Nov 17 2023 Anton Novojilov <andy@essentialkaos.com> - 1.6.1-0
- [cli] [sync] Minor UI improvements
- [cli] Added check for newer versions
- [cli] Fixed minor bug with coloring Redis commands in tips

* Tue Nov 07 2023 Anton Novojilov <andy@essentialkaos.com> - 1.6.0-0
- [cli] Print extra info using 'list' command with -x/--extra option
- [cli] Added warning about unknown commands on minion/sentinel nodes
- [cli] Improved "replication" command output
- [sync] Added "bye" API command to unregister minion and sentinel nodes on stop
- [sync] Added info about lag to client info
- [sync] Better request validation
- [sync] Improved error logging
- [cli] [sync] Code refactoring

* Mon Nov 06 2023 Anton Novojilov <andy@essentialkaos.com> - 1.5.1-0
- [cli] Tips fixes

* Sat Oct 21 2023 Anton Novojilov <andy@essentialkaos.com> - 1.5.0-0
- [cli] Added protip tips
- [cli] Added user-specific preferences
- [cli] Improved checks before role changing
- [cli] Minor fixes

* Tue Oct 17 2023 Anton Novojilov <andy@essentialkaos.com> - 1.4.3-0
- [cli] Fixed maintenance mode notification position
- Removed outdated option from configuration file

* Tue Oct 17 2023 Anton Novojilov <andy@essentialkaos.com> - 1.4.2-0
- [cli] Improved support info output
- [cli] Improved full-text search results highlighting
- [sync] Added extended check for master IP
- [sync] Improved configuration validation
- Dependencies update

* Fri Oct 13 2023 Anton Novojilov <andy@essentialkaos.com> - 1.4.1-0
- [cli] Improved full-text search using 'list' command

* Fri Oct 13 2023 Anton Novojilov <andy@essentialkaos.com> - 1.4.0-0
- [cli] Changed syntax for listing instances with tags
- [cli] Added full-text search to 'list' command
- [cli] Changed tags rendering format
- [cli] Fixed bug with handling 'MONITOR' command
- Dependencies update

* Fri Sep 29 2023 Anton Novojilov <andy@essentialkaos.com> - 1.3.1-0
- [cli] Added autocorrect of section names for the 'info' command

* Fri Sep 29 2023 Anton Novojilov <andy@essentialkaos.com> - 1.3.0-0
- [cli] Added more filters to 'list' command
- [cli] Verbose log messages about meta editing
- Fixed sync user credentials rendering for standby instances

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
- [cli] Improved instance listing filtering
- [cli] Improved actions logging
- [sync] 'max-init-sync-wait' deprecated
- [cli] Fixed raw output from 'top' command for long numbers
- [cli] Fixed command execution with 'cli'
- [cli] Fixed password check using password variations

* Tue Aug 22 2023 Anton Novojilov <andy@essentialkaos.com> - 1.1.0-0
- Added info about RDS to templates payload
- Added instance storage data to templates payload

* Mon Aug 21 2023 Anton Novojilov <andy@essentialkaos.com> - 1.0.0-0
- First public release
