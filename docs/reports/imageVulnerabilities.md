# imageVulnerabilities

```json
{
    "reportGenerated": "2024-09-05T17:10:08.669122025-06:00",
    "reportType": "imageVulnerabilities",
    "image": "docker.io/library/nginx:latest",
    "report": {
        "imageCreated": "2024-08-14T21:31:12Z",
        "imageIssues": {
            "total": {
                "unknown": 0,
                "low": 86,
                "medium": 38,
                "high": 16,
                "critical": 5
            },
            "misconfigurations": {
                "unknown": 0,
                "low": 0,
                "medium": 0,
                "high": 0,
                "critical": 0
            },
            "vulnerabilities": {
                "unknown": 0,
                "low": 86,
                "medium": 38,
                "high": 16,
                "critical": 5,
                "vulnerabilities": [
                    {
                        "registry": "CVE-2011-3374",
                        "severity": "LOW",
                        "pkgID": "apt@2.6.1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2011-3374",
                        "title": "It was found that apt-key in apt, all versions, do not correctly valid ...",
                        "description": "It was found that apt-key in apt, all versions, do not correctly validate gpg keys with the master keyring, leading to a potential man-in-the-middle attack.",
                        "nvdV3Score": 3.7,
                        "publishedDate": "2019-11-26T00:15:11.03Z"
                    },
                    {
                        "registry": "TEMP-0841856-B18BAF",
                        "severity": "LOW",
                        "pkgID": "bash@5.2.15-2+b7",
                        "primaryURL": "https://security-tracker.debian.org/tracker/TEMP-0841856-B18BAF",
                        "title": "[Privilege escalation possible to other user than root]",
                        "description": "",
                        "nvdV3Score": 0,
                        "publishedDate": null
                    },
                    {
                        "registry": "CVE-2022-0563",
                        "severity": "LOW",
                        "pkgID": "bsdutils@1:2.38.1-5+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2022-0563",
                        "title": "util-linux: partial disclosure of arbitrary files in chfn and chsh when compiled with libreadline",
                        "description": "A flaw was found in the util-linux chfn and chsh utilities when compiled with Readline support. The Readline library uses an \"INPUTRC\" environment variable to get a path to the library config file. When the library cannot parse the specified file, it prints an error message containing data from the file. This flaw allows an unprivileged user to read root-owned files, potentially leading to privilege escalation. This flaw affects util-linux versions prior to 2.37.4.",
                        "nvdV3Score": 5.5,
                        "publishedDate": "2022-02-21T19:15:08.393Z"
                    },
                    {
                        "registry": "CVE-2016-2781",
                        "severity": "LOW",
                        "pkgID": "coreutils@9.1-1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2016-2781",
                        "title": "coreutils: Non-privileged session can escape to the parent session in chroot",
                        "description": "chroot in GNU coreutils, when used with --userspec, allows local users to escape to the parent session via a crafted TIOCSTI ioctl call, which pushes characters to the terminal's input buffer.",
                        "nvdV3Score": 6.5,
                        "publishedDate": "2017-02-07T15:59:00.333Z"
                    },
                    {
                        "registry": "CVE-2017-18018",
                        "severity": "LOW",
                        "pkgID": "coreutils@9.1-1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2017-18018",
                        "title": "coreutils: race condition vulnerability in chown and chgrp",
                        "description": "In GNU Coreutils through 8.29, chown-core.c in chown and chgrp does not prevent replacement of a plain file with a symlink during use of the POSIX \"-R -L\" options, which allows local users to modify the ownership of arbitrary files by leveraging a race condition.",
                        "nvdV3Score": 4.7,
                        "publishedDate": "2018-01-04T04:29:00.19Z"
                    },
                    {
                        "registry": "CVE-2024-2379",
                        "severity": "LOW",
                        "pkgID": "curl@7.88.1-10+deb12u7",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-2379",
                        "title": "curl: QUIC certificate check bypass with wolfSSL",
                        "description": "libcurl skips the certificate verification for a QUIC connection under certain conditions, when built to use wolfSSL. If told to use an unknown/bad cipher or curve, the error path accidentally skips the verification and returns OK, thus ignoring any certificate problems.",
                        "nvdV3Score": 0,
                        "publishedDate": "2024-03-27T08:15:41.23Z"
                    },
                    {
                        "registry": "CVE-2023-4039",
                        "severity": "MEDIUM",
                        "pkgID": "gcc-12-base@12.2.0-14",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-4039",
                        "title": "gcc: -fstack-protector fails to guard dynamic stack allocations on ARM64",
                        "description": "\n\n**DISPUTED**A failure in the -fstack-protector feature in GCC-based toolchains \nthat target AArch64 allows an attacker to exploit an existing buffer \noverflow in dynamically-sized local variables in your application \nwithout this being detected. This stack-protector failure only applies \nto C99-style dynamically-sized local variables or those created using \nalloca(). The stack-protector operates as intended for statically-sized \nlocal variables.\n\nThe default behavior when the stack-protector \ndetects an overflow is to terminate your application, resulting in \ncontrolled loss of availability. An attacker who can exploit a buffer \noverflow without triggering the stack-protector might be able to change \nprogram flow control to cause an uncontrolled loss of availability or to\n go further and affect confidentiality or integrity. NOTE: The GCC project argues that this is a missed hardening bug and not a vulnerability by itself.\n\n\n\n\n\n",
                        "nvdV3Score": 4.8,
                        "publishedDate": "2023-09-13T09:15:15.69Z"
                    },
                    {
                        "registry": "CVE-2022-27943",
                        "severity": "LOW",
                        "pkgID": "gcc-12-base@12.2.0-14",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2022-27943",
                        "title": "binutils: libiberty/rust-demangle.c in GNU GCC 11.2 allows stack exhaustion in demangle_const",
                        "description": "libiberty/rust-demangle.c in GNU GCC 11.2 allows stack consumption in demangle_const, as demonstrated by nm-new.",
                        "nvdV3Score": 5.5,
                        "publishedDate": "2022-03-26T13:15:07.9Z"
                    },
                    {
                        "registry": "CVE-2022-3219",
                        "severity": "LOW",
                        "pkgID": "gpgv@2.2.40-1.1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2022-3219",
                        "title": "gnupg: denial of service issue (resource consumption) using compressed packets",
                        "description": "GnuPG can be made to spin on a relatively small input by (for example) crafting a public key with thousands of signatures attached, compressed down to just a few KB.",
                        "nvdV3Score": 3.3,
                        "publishedDate": "2023-02-23T20:15:12.393Z"
                    },
                    {
                        "registry": "CVE-2023-6879",
                        "severity": "CRITICAL",
                        "pkgID": "libaom3@3.6.0-1+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-6879",
                        "title": "aom: heap-buffer-overflow on frame size change",
                        "description": "Increasing the resolution of video frames, while performing a multi-threaded encode, can result in a heap overflow in av1_loop_restoration_dealloc().\n\n",
                        "nvdV3Score": 9.8,
                        "publishedDate": "2023-12-27T23:15:07.53Z"
                    },
                    {
                        "registry": "CVE-2023-39616",
                        "severity": "HIGH",
                        "pkgID": "libaom3@3.6.0-1+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-39616",
                        "title": "AOMedia v3.0.0 to v3.5.0 was discovered to contain an invalid read mem ...",
                        "description": "AOMedia v3.0.0 to v3.5.0 was discovered to contain an invalid read memory access via the component assign_frame_buffer_p in av1/common/av1_common_int.h.",
                        "nvdV3Score": 7.5,
                        "publishedDate": "2023-08-29T17:15:12.633Z"
                    },
                    {
                        "registry": "CVE-2011-3374",
                        "severity": "LOW",
                        "pkgID": "libapt-pkg6.0@2.6.1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2011-3374",
                        "title": "It was found that apt-key in apt, all versions, do not correctly valid ...",
                        "description": "It was found that apt-key in apt, all versions, do not correctly validate gpg keys with the master keyring, leading to a potential man-in-the-middle attack.",
                        "nvdV3Score": 3.7,
                        "publishedDate": "2019-11-26T00:15:11.03Z"
                    },
                    {
                        "registry": "CVE-2022-0563",
                        "severity": "LOW",
                        "pkgID": "libblkid1@2.38.1-5+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2022-0563",
                        "title": "util-linux: partial disclosure of arbitrary files in chfn and chsh when compiled with libreadline",
                        "description": "A flaw was found in the util-linux chfn and chsh utilities when compiled with Readline support. The Readline library uses an \"INPUTRC\" environment variable to get a path to the library config file. When the library cannot parse the specified file, it prints an error message containing data from the file. This flaw allows an unprivileged user to read root-owned files, potentially leading to privilege escalation. This flaw affects util-linux versions prior to 2.37.4.",
                        "nvdV3Score": 5.5,
                        "publishedDate": "2022-02-21T19:15:08.393Z"
                    },
                    {
                        "registry": "CVE-2010-4756",
                        "severity": "LOW",
                        "pkgID": "libc-bin@2.36-9+deb12u8",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2010-4756",
                        "title": "glibc: glob implementation can cause excessive CPU and memory consumption due to crafted glob expressions",
                        "description": "The glob implementation in the GNU C Library (aka glibc or libc6) allows remote authenticated users to cause a denial of service (CPU and memory consumption) via crafted glob expressions that do not match any pathnames, as demonstrated by glob expressions in STAT commands to an FTP daemon, a different vulnerability than CVE-2010-2632.",
                        "nvdV3Score": 0,
                        "publishedDate": "2011-03-02T20:00:01.037Z"
                    },
                    {
                        "registry": "CVE-2018-20796",
                        "severity": "LOW",
                        "pkgID": "libc-bin@2.36-9+deb12u8",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2018-20796",
                        "title": "glibc: uncontrolled recursion in function check_dst_limits_calc_pos_1 in posix/regexec.c",
                        "description": "In the GNU C Library (aka glibc or libc6) through 2.29, check_dst_limits_calc_pos_1 in posix/regexec.c has Uncontrolled Recursion, as demonstrated by '(\\227|)(\\\\1\\\\1|t1|\\\\\\2537)+' in grep.",
                        "nvdV3Score": 7.5,
                        "publishedDate": "2019-02-26T02:29:00.45Z"
                    },
                    {
                        "registry": "CVE-2019-1010022",
                        "severity": "LOW",
                        "pkgID": "libc-bin@2.36-9+deb12u8",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2019-1010022",
                        "title": "glibc: stack guard protection bypass",
                        "description": "GNU Libc current is affected by: Mitigation bypass. The impact is: Attacker may bypass stack guard protection. The component is: nptl. The attack vector is: Exploit stack buffer overflow vulnerability and use this bypass vulnerability to bypass stack guard. NOTE: Upstream comments indicate \"this is being treated as a non-security bug and no real threat.",
                        "nvdV3Score": 9.8,
                        "publishedDate": "2019-07-15T04:15:13.317Z"
                    },
                    {
                        "registry": "CVE-2019-1010023",
                        "severity": "LOW",
                        "pkgID": "libc-bin@2.36-9+deb12u8",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2019-1010023",
                        "title": "glibc: running ldd on malicious ELF leads to code execution because of wrong size computation",
                        "description": "GNU Libc current is affected by: Re-mapping current loaded library with malicious ELF file. The impact is: In worst case attacker may evaluate privileges. The component is: libld. The attack vector is: Attacker sends 2 ELF files to victim and asks to run ldd on it. ldd execute code. NOTE: Upstream comments indicate \"this is being treated as a non-security bug and no real threat.",
                        "nvdV3Score": 8.8,
                        "publishedDate": "2019-07-15T04:15:13.397Z"
                    },
                    {
                        "registry": "CVE-2019-1010024",
                        "severity": "LOW",
                        "pkgID": "libc-bin@2.36-9+deb12u8",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2019-1010024",
                        "title": "glibc: ASLR bypass using cache of thread stack and heap",
                        "description": "GNU Libc current is affected by: Mitigation bypass. The impact is: Attacker may bypass ASLR using cache of thread stack and heap. The component is: glibc. NOTE: Upstream comments indicate \"this is being treated as a non-security bug and no real threat.",
                        "nvdV3Score": 5.3,
                        "publishedDate": "2019-07-15T04:15:13.473Z"
                    },
                    {
                        "registry": "CVE-2019-1010025",
                        "severity": "LOW",
                        "pkgID": "libc-bin@2.36-9+deb12u8",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2019-1010025",
                        "title": "glibc: information disclosure of heap addresses of pthread_created thread",
                        "description": "GNU Libc current is affected by: Mitigation bypass. The impact is: Attacker may guess the heap addresses of pthread_created thread. The component is: glibc. NOTE: the vendor's position is \"ASLR bypass itself is not a vulnerability.",
                        "nvdV3Score": 5.3,
                        "publishedDate": "2019-07-15T04:15:13.537Z"
                    },
                    {
                        "registry": "CVE-2019-9192",
                        "severity": "LOW",
                        "pkgID": "libc-bin@2.36-9+deb12u8",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2019-9192",
                        "title": "glibc: uncontrolled recursion in function check_dst_limits_calc_pos_1 in posix/regexec.c",
                        "description": "In the GNU C Library (aka glibc or libc6) through 2.29, check_dst_limits_calc_pos_1 in posix/regexec.c has Uncontrolled Recursion, as demonstrated by '(|)(\\\\1\\\\1)*' in grep, a different issue than CVE-2018-20796. NOTE: the software maintainer disputes that this is a vulnerability because the behavior occurs only with a crafted pattern",
                        "nvdV3Score": 7.5,
                        "publishedDate": "2019-02-26T18:29:00.34Z"
                    },
                    {
                        "registry": "CVE-2010-4756",
                        "severity": "LOW",
                        "pkgID": "libc6@2.36-9+deb12u8",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2010-4756",
                        "title": "glibc: glob implementation can cause excessive CPU and memory consumption due to crafted glob expressions",
                        "description": "The glob implementation in the GNU C Library (aka glibc or libc6) allows remote authenticated users to cause a denial of service (CPU and memory consumption) via crafted glob expressions that do not match any pathnames, as demonstrated by glob expressions in STAT commands to an FTP daemon, a different vulnerability than CVE-2010-2632.",
                        "nvdV3Score": 0,
                        "publishedDate": "2011-03-02T20:00:01.037Z"
                    },
                    {
                        "registry": "CVE-2018-20796",
                        "severity": "LOW",
                        "pkgID": "libc6@2.36-9+deb12u8",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2018-20796",
                        "title": "glibc: uncontrolled recursion in function check_dst_limits_calc_pos_1 in posix/regexec.c",
                        "description": "In the GNU C Library (aka glibc or libc6) through 2.29, check_dst_limits_calc_pos_1 in posix/regexec.c has Uncontrolled Recursion, as demonstrated by '(\\227|)(\\\\1\\\\1|t1|\\\\\\2537)+' in grep.",
                        "nvdV3Score": 7.5,
                        "publishedDate": "2019-02-26T02:29:00.45Z"
                    },
                    {
                        "registry": "CVE-2019-1010022",
                        "severity": "LOW",
                        "pkgID": "libc6@2.36-9+deb12u8",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2019-1010022",
                        "title": "glibc: stack guard protection bypass",
                        "description": "GNU Libc current is affected by: Mitigation bypass. The impact is: Attacker may bypass stack guard protection. The component is: nptl. The attack vector is: Exploit stack buffer overflow vulnerability and use this bypass vulnerability to bypass stack guard. NOTE: Upstream comments indicate \"this is being treated as a non-security bug and no real threat.",
                        "nvdV3Score": 9.8,
                        "publishedDate": "2019-07-15T04:15:13.317Z"
                    },
                    {
                        "registry": "CVE-2019-1010023",
                        "severity": "LOW",
                        "pkgID": "libc6@2.36-9+deb12u8",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2019-1010023",
                        "title": "glibc: running ldd on malicious ELF leads to code execution because of wrong size computation",
                        "description": "GNU Libc current is affected by: Re-mapping current loaded library with malicious ELF file. The impact is: In worst case attacker may evaluate privileges. The component is: libld. The attack vector is: Attacker sends 2 ELF files to victim and asks to run ldd on it. ldd execute code. NOTE: Upstream comments indicate \"this is being treated as a non-security bug and no real threat.",
                        "nvdV3Score": 8.8,
                        "publishedDate": "2019-07-15T04:15:13.397Z"
                    },
                    {
                        "registry": "CVE-2019-1010024",
                        "severity": "LOW",
                        "pkgID": "libc6@2.36-9+deb12u8",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2019-1010024",
                        "title": "glibc: ASLR bypass using cache of thread stack and heap",
                        "description": "GNU Libc current is affected by: Mitigation bypass. The impact is: Attacker may bypass ASLR using cache of thread stack and heap. The component is: glibc. NOTE: Upstream comments indicate \"this is being treated as a non-security bug and no real threat.",
                        "nvdV3Score": 5.3,
                        "publishedDate": "2019-07-15T04:15:13.473Z"
                    },
                    {
                        "registry": "CVE-2019-1010025",
                        "severity": "LOW",
                        "pkgID": "libc6@2.36-9+deb12u8",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2019-1010025",
                        "title": "glibc: information disclosure of heap addresses of pthread_created thread",
                        "description": "GNU Libc current is affected by: Mitigation bypass. The impact is: Attacker may guess the heap addresses of pthread_created thread. The component is: glibc. NOTE: the vendor's position is \"ASLR bypass itself is not a vulnerability.",
                        "nvdV3Score": 5.3,
                        "publishedDate": "2019-07-15T04:15:13.537Z"
                    },
                    {
                        "registry": "CVE-2019-9192",
                        "severity": "LOW",
                        "pkgID": "libc6@2.36-9+deb12u8",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2019-9192",
                        "title": "glibc: uncontrolled recursion in function check_dst_limits_calc_pos_1 in posix/regexec.c",
                        "description": "In the GNU C Library (aka glibc or libc6) through 2.29, check_dst_limits_calc_pos_1 in posix/regexec.c has Uncontrolled Recursion, as demonstrated by '(|)(\\\\1\\\\1)*' in grep, a different issue than CVE-2018-20796. NOTE: the software maintainer disputes that this is a vulnerability because the behavior occurs only with a crafted pattern",
                        "nvdV3Score": 7.5,
                        "publishedDate": "2019-02-26T18:29:00.34Z"
                    },
                    {
                        "registry": "CVE-2024-2379",
                        "severity": "LOW",
                        "pkgID": "libcurl4@7.88.1-10+deb12u7",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-2379",
                        "title": "curl: QUIC certificate check bypass with wolfSSL",
                        "description": "libcurl skips the certificate verification for a QUIC connection under certain conditions, when built to use wolfSSL. If told to use an unknown/bad cipher or curve, the error path accidentally skips the verification and returns OK, thus ignoring any certificate problems.",
                        "nvdV3Score": 0,
                        "publishedDate": "2024-03-27T08:15:41.23Z"
                    },
                    {
                        "registry": "CVE-2023-32570",
                        "severity": "MEDIUM",
                        "pkgID": "libdav1d6@1.0.0-2+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-32570",
                        "title": "VideoLAN dav1d before 1.2.0 has a thread_task.c race condition that ca ...",
                        "description": "VideoLAN dav1d before 1.2.0 has a thread_task.c race condition that can lead to an application crash, related to dav1d_decode_frame_exit.",
                        "nvdV3Score": 5.9,
                        "publishedDate": "2023-05-10T05:15:12.19Z"
                    },
                    {
                        "registry": "CVE-2023-51792",
                        "severity": "MEDIUM",
                        "pkgID": "libde265-0@1.0.11-1+deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-51792",
                        "title": "Buffer Overflow vulnerability in libde265 v1.0.12 allows a local attac ...",
                        "description": "Buffer Overflow vulnerability in libde265 v1.0.12 allows a local attacker to cause a denial of service via the allocation size exceeding the maximum supported size of 0x10000000000.",
                        "nvdV3Score": 0,
                        "publishedDate": "2024-04-19T17:15:52.24Z"
                    },
                    {
                        "registry": "CVE-2024-38949",
                        "severity": "MEDIUM",
                        "pkgID": "libde265-0@1.0.11-1+deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-38949",
                        "title": "Heap Buffer Overflow vulnerability in Libde265 v1.0.15 allows attacker ...",
                        "description": "Heap Buffer Overflow vulnerability in Libde265 v1.0.15 allows attackers to crash the application via crafted payload to display444as420 function at sdl.cc",
                        "nvdV3Score": 0,
                        "publishedDate": "2024-06-26T20:15:16.263Z"
                    },
                    {
                        "registry": "CVE-2024-38950",
                        "severity": "MEDIUM",
                        "pkgID": "libde265-0@1.0.11-1+deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-38950",
                        "title": "Heap Buffer Overflow vulnerability in Libde265 v1.0.15 allows attacker ...",
                        "description": "Heap Buffer Overflow vulnerability in Libde265 v1.0.15 allows attackers to crash the application via crafted payload to __interceptor_memcpy function.",
                        "nvdV3Score": 0,
                        "publishedDate": "2024-06-26T20:15:16.367Z"
                    },
                    {
                        "registry": "CVE-2024-45490",
                        "severity": "CRITICAL",
                        "pkgID": "libexpat1@2.5.0-1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-45490",
                        "title": "libexpat: Negative Length Parsing Vulnerability in libexpat",
                        "description": "An issue was discovered in libexpat before 2.6.3. xmlparse.c does not reject a negative length for XML_ParseBuffer.",
                        "nvdV3Score": 9.8,
                        "publishedDate": "2024-08-30T03:15:03.757Z"
                    },
                    {
                        "registry": "CVE-2024-45491",
                        "severity": "CRITICAL",
                        "pkgID": "libexpat1@2.5.0-1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-45491",
                        "title": "libexpat: Integer Overflow or Wraparound",
                        "description": "An issue was discovered in libexpat before 2.6.3. dtdCopy in xmlparse.c can have an integer overflow for nDefaultAtts on 32-bit platforms (where UINT_MAX equals SIZE_MAX).",
                        "nvdV3Score": 9.8,
                        "publishedDate": "2024-08-30T03:15:03.85Z"
                    },
                    {
                        "registry": "CVE-2024-45492",
                        "severity": "CRITICAL",
                        "pkgID": "libexpat1@2.5.0-1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-45492",
                        "title": "libexpat: integer overflow",
                        "description": "An issue was discovered in libexpat before 2.6.3. nextScaffoldPart in xmlparse.c can have an integer overflow for m_groupSize on 32-bit platforms (where UINT_MAX equals SIZE_MAX).",
                        "nvdV3Score": 9.8,
                        "publishedDate": "2024-08-30T03:15:03.93Z"
                    },
                    {
                        "registry": "CVE-2023-52425",
                        "severity": "HIGH",
                        "pkgID": "libexpat1@2.5.0-1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-52425",
                        "title": "expat: parsing large tokens can trigger a denial of service",
                        "description": "libexpat through 2.5.0 allows a denial of service (resource consumption) because many full reparsings are required in the case of a large token for which multiple buffer fills are needed.",
                        "nvdV3Score": 7.5,
                        "publishedDate": "2024-02-04T20:15:46.063Z"
                    },
                    {
                        "registry": "CVE-2023-52426",
                        "severity": "LOW",
                        "pkgID": "libexpat1@2.5.0-1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-52426",
                        "title": "expat: recursive XML entity expansion vulnerability",
                        "description": "libexpat through 2.5.0 allows recursive XML Entity Expansion if XML_DTD is undefined at compile time.",
                        "nvdV3Score": 5.5,
                        "publishedDate": "2024-02-04T20:15:46.12Z"
                    },
                    {
                        "registry": "CVE-2024-28757",
                        "severity": "LOW",
                        "pkgID": "libexpat1@2.5.0-1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-28757",
                        "title": "expat: XML Entity Expansion",
                        "description": "libexpat through 2.6.1 allows an XML Entity Expansion attack when there is isolated use of external parsers (created via XML_ExternalEntityParserCreate).",
                        "nvdV3Score": 0,
                        "publishedDate": "2024-03-10T05:15:06.57Z"
                    },
                    {
                        "registry": "CVE-2023-4039",
                        "severity": "MEDIUM",
                        "pkgID": "libgcc-s1@12.2.0-14",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-4039",
                        "title": "gcc: -fstack-protector fails to guard dynamic stack allocations on ARM64",
                        "description": "\n\n**DISPUTED**A failure in the -fstack-protector feature in GCC-based toolchains \nthat target AArch64 allows an attacker to exploit an existing buffer \noverflow in dynamically-sized local variables in your application \nwithout this being detected. This stack-protector failure only applies \nto C99-style dynamically-sized local variables or those created using \nalloca(). The stack-protector operates as intended for statically-sized \nlocal variables.\n\nThe default behavior when the stack-protector \ndetects an overflow is to terminate your application, resulting in \ncontrolled loss of availability. An attacker who can exploit a buffer \noverflow without triggering the stack-protector might be able to change \nprogram flow control to cause an uncontrolled loss of availability or to\n go further and affect confidentiality or integrity. NOTE: The GCC project argues that this is a missed hardening bug and not a vulnerability by itself.\n\n\n\n\n\n",
                        "nvdV3Score": 4.8,
                        "publishedDate": "2023-09-13T09:15:15.69Z"
                    },
                    {
                        "registry": "CVE-2022-27943",
                        "severity": "LOW",
                        "pkgID": "libgcc-s1@12.2.0-14",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2022-27943",
                        "title": "binutils: libiberty/rust-demangle.c in GNU GCC 11.2 allows stack exhaustion in demangle_const",
                        "description": "libiberty/rust-demangle.c in GNU GCC 11.2 allows stack consumption in demangle_const, as demonstrated by nm-new.",
                        "nvdV3Score": 5.5,
                        "publishedDate": "2022-03-26T13:15:07.9Z"
                    },
                    {
                        "registry": "CVE-2024-2236",
                        "severity": "MEDIUM",
                        "pkgID": "libgcrypt20@1.10.1-3",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-2236",
                        "title": "libgcrypt: vulnerable to Marvin Attack",
                        "description": "A timing-based side-channel flaw was found in libgcrypt's RSA implementation. This issue may allow a remote attacker to initiate a Bleichenbacher-style attack, which can lead to the decryption of RSA ciphertexts.",
                        "nvdV3Score": 0,
                        "publishedDate": "2024-03-06T22:15:57.977Z"
                    },
                    {
                        "registry": "CVE-2018-6829",
                        "severity": "LOW",
                        "pkgID": "libgcrypt20@1.10.1-3",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2018-6829",
                        "title": "libgcrypt: ElGamal implementation doesn't have semantic security due to incorrectly encoded plaintexts possibly allowing to obtain sensitive information",
                        "description": "cipher/elgamal.c in Libgcrypt through 1.8.2, when used to encrypt messages directly, improperly encodes plaintexts, which allows attackers to obtain sensitive information by reading ciphertext data (i.e., it does not have semantic security in face of a ciphertext-only attack). The Decisional Diffie-Hellman (DDH) assumption does not hold for Libgcrypt's ElGamal implementation.",
                        "nvdV3Score": 7.5,
                        "publishedDate": "2018-02-07T23:29:01.703Z"
                    },
                    {
                        "registry": "CVE-2011-3389",
                        "severity": "LOW",
                        "pkgID": "libgnutls30@3.7.9-2+deb12u3",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2011-3389",
                        "title": "HTTPS: block-wise chosen-plaintext attack against SSL/TLS (BEAST)",
                        "description": "The SSL protocol, as used in certain configurations in Microsoft Windows and Microsoft Internet Explorer, Mozilla Firefox, Google Chrome, Opera, and other products, encrypts data by using CBC mode with chained initialization vectors, which allows man-in-the-middle attackers to obtain plaintext HTTP headers via a blockwise chosen-boundary attack (BCBA) on an HTTPS session, in conjunction with JavaScript code that uses (1) the HTML5 WebSocket API, (2) the Java URLConnection API, or (3) the Silverlight WebClient API, aka a \"BEAST\" attack.",
                        "nvdV3Score": 0,
                        "publishedDate": "2011-09-06T19:55:03.197Z"
                    },
                    {
                        "registry": "CVE-2024-26462",
                        "severity": "HIGH",
                        "pkgID": "libgssapi-krb5-2@1.20.1-2+deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-26462",
                        "title": "krb5: Memory leak at /krb5/src/kdc/ndr.c",
                        "description": "Kerberos 5 (aka krb5) 1.21.2 contains a memory leak vulnerability in /krb5/src/kdc/ndr.c.",
                        "nvdV3Score": 0,
                        "publishedDate": "2024-02-29T01:44:18.857Z"
                    },
                    {
                        "registry": "CVE-2024-26458",
                        "severity": "MEDIUM",
                        "pkgID": "libgssapi-krb5-2@1.20.1-2+deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-26458",
                        "title": "krb5: Memory leak at /krb5/src/lib/rpc/pmap_rmt.c",
                        "description": "Kerberos 5 (aka krb5) 1.21.2 contains a memory leak in /krb5/src/lib/rpc/pmap_rmt.c.",
                        "nvdV3Score": 0,
                        "publishedDate": "2024-02-29T01:44:18.78Z"
                    },
                    {
                        "registry": "CVE-2024-26461",
                        "severity": "MEDIUM",
                        "pkgID": "libgssapi-krb5-2@1.20.1-2+deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-26461",
                        "title": "krb5: Memory leak at /krb5/src/lib/gssapi/krb5/k5sealv3.c",
                        "description": "Kerberos 5 (aka krb5) 1.21.2 contains a memory leak vulnerability in /krb5/src/lib/gssapi/krb5/k5sealv3.c.",
                        "nvdV3Score": 0,
                        "publishedDate": "2024-02-29T01:44:18.82Z"
                    },
                    {
                        "registry": "CVE-2018-5709",
                        "severity": "LOW",
                        "pkgID": "libgssapi-krb5-2@1.20.1-2+deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2018-5709",
                        "title": "krb5: integer overflow in dbentry-\u003en_key_data in kadmin/dbutil/dump.c",
                        "description": "An issue was discovered in MIT Kerberos 5 (aka krb5) through 1.16. There is a variable \"dbentry-\u003en_key_data\" in kadmin/dbutil/dump.c that can store 16-bit data but unknowingly the developer has assigned a \"u4\" variable to it, which is for 32-bit data. An attacker can use this vulnerability to affect other artifacts of the database as we know that a Kerberos database dump file contains trusted data.",
                        "nvdV3Score": 7.5,
                        "publishedDate": "2018-01-16T09:29:00.5Z"
                    },
                    {
                        "registry": "CVE-2023-49460",
                        "severity": "HIGH",
                        "pkgID": "libheif1@1.15.1-1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-49460",
                        "title": "libheif v1.17.5 was discovered to contain a segmentation violation via ...",
                        "description": "libheif v1.17.5 was discovered to contain a segmentation violation via the function UncompressedImageCodec::decode_uncompressed_image.",
                        "nvdV3Score": 8.8,
                        "publishedDate": "2023-12-07T20:15:38.14Z"
                    },
                    {
                        "registry": "CVE-2023-49462",
                        "severity": "HIGH",
                        "pkgID": "libheif1@1.15.1-1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-49462",
                        "title": "libheif v1.17.5 was discovered to contain a segmentation violation via ...",
                        "description": "libheif v1.17.5 was discovered to contain a segmentation violation via the component /libheif/exif.cc.",
                        "nvdV3Score": 8.8,
                        "publishedDate": "2023-12-07T20:15:38.19Z"
                    },
                    {
                        "registry": "CVE-2023-49463",
                        "severity": "HIGH",
                        "pkgID": "libheif1@1.15.1-1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-49463",
                        "title": "libheif v1.17.5 was discovered to contain a segmentation violation via ...",
                        "description": "libheif v1.17.5 was discovered to contain a segmentation violation via the function find_exif_tag at /libheif/exif.cc.",
                        "nvdV3Score": 8.8,
                        "publishedDate": "2023-12-07T20:15:38.26Z"
                    },
                    {
                        "registry": "CVE-2023-49464",
                        "severity": "HIGH",
                        "pkgID": "libheif1@1.15.1-1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-49464",
                        "title": "libheif v1.17.5 was discovered to contain a segmentation violation via ...",
                        "description": "libheif v1.17.5 was discovered to contain a segmentation violation via the function UncompressedImageCodec::get_luma_bits_per_pixel_from_configuration_unci.",
                        "nvdV3Score": 8.8,
                        "publishedDate": "2023-12-07T20:15:38.32Z"
                    },
                    {
                        "registry": "CVE-2023-29659",
                        "severity": "MEDIUM",
                        "pkgID": "libheif1@1.15.1-1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-29659",
                        "title": "A Segmentation fault caused by a floating point exception exists in li ...",
                        "description": "A Segmentation fault caused by a floating point exception exists in libheif 1.15.1 using crafted heif images via the heif::Fraction::round() function in box.cc, which causes a denial of service.",
                        "nvdV3Score": 6.5,
                        "publishedDate": "2023-05-05T16:15:09.387Z"
                    },
                    {
                        "registry": "CVE-2024-25269",
                        "severity": "LOW",
                        "pkgID": "libheif1@1.15.1-1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-25269",
                        "title": "libheif \u003c= 1.17.6 contains a memory leak in the function JpegEncoder:: ...",
                        "description": "libheif \u003c= 1.17.6 contains a memory leak in the function JpegEncoder::Encode. This flaw allows an attacker to cause a denial of service attack.",
                        "nvdV3Score": 0,
                        "publishedDate": "2024-03-05T01:15:06.78Z"
                    },
                    {
                        "registry": "CVE-2017-9937",
                        "severity": "LOW",
                        "pkgID": "libjbig0@2.1-6.1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2017-9937",
                        "title": "libtiff: memory malloc failure in tif_jbig.c could cause DOS.",
                        "description": "In LibTIFF 4.0.8, there is a memory malloc failure in tif_jbig.c. A crafted TIFF document can lead to an abort resulting in a remote denial of service attack.",
                        "nvdV3Score": 6.5,
                        "publishedDate": "2017-06-26T12:29:00.25Z"
                    },
                    {
                        "registry": "CVE-2024-26462",
                        "severity": "HIGH",
                        "pkgID": "libk5crypto3@1.20.1-2+deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-26462",
                        "title": "krb5: Memory leak at /krb5/src/kdc/ndr.c",
                        "description": "Kerberos 5 (aka krb5) 1.21.2 contains a memory leak vulnerability in /krb5/src/kdc/ndr.c.",
                        "nvdV3Score": 0,
                        "publishedDate": "2024-02-29T01:44:18.857Z"
                    },
                    {
                        "registry": "CVE-2024-26458",
                        "severity": "MEDIUM",
                        "pkgID": "libk5crypto3@1.20.1-2+deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-26458",
                        "title": "krb5: Memory leak at /krb5/src/lib/rpc/pmap_rmt.c",
                        "description": "Kerberos 5 (aka krb5) 1.21.2 contains a memory leak in /krb5/src/lib/rpc/pmap_rmt.c.",
                        "nvdV3Score": 0,
                        "publishedDate": "2024-02-29T01:44:18.78Z"
                    },
                    {
                        "registry": "CVE-2024-26461",
                        "severity": "MEDIUM",
                        "pkgID": "libk5crypto3@1.20.1-2+deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-26461",
                        "title": "krb5: Memory leak at /krb5/src/lib/gssapi/krb5/k5sealv3.c",
                        "description": "Kerberos 5 (aka krb5) 1.21.2 contains a memory leak vulnerability in /krb5/src/lib/gssapi/krb5/k5sealv3.c.",
                        "nvdV3Score": 0,
                        "publishedDate": "2024-02-29T01:44:18.82Z"
                    },
                    {
                        "registry": "CVE-2018-5709",
                        "severity": "LOW",
                        "pkgID": "libk5crypto3@1.20.1-2+deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2018-5709",
                        "title": "krb5: integer overflow in dbentry-\u003en_key_data in kadmin/dbutil/dump.c",
                        "description": "An issue was discovered in MIT Kerberos 5 (aka krb5) through 1.16. There is a variable \"dbentry-\u003en_key_data\" in kadmin/dbutil/dump.c that can store 16-bit data but unknowingly the developer has assigned a \"u4\" variable to it, which is for 32-bit data. An attacker can use this vulnerability to affect other artifacts of the database as we know that a Kerberos database dump file contains trusted data.",
                        "nvdV3Score": 7.5,
                        "publishedDate": "2018-01-16T09:29:00.5Z"
                    },
                    {
                        "registry": "CVE-2024-26462",
                        "severity": "HIGH",
                        "pkgID": "libkrb5-3@1.20.1-2+deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-26462",
                        "title": "krb5: Memory leak at /krb5/src/kdc/ndr.c",
                        "description": "Kerberos 5 (aka krb5) 1.21.2 contains a memory leak vulnerability in /krb5/src/kdc/ndr.c.",
                        "nvdV3Score": 0,
                        "publishedDate": "2024-02-29T01:44:18.857Z"
                    },
                    {
                        "registry": "CVE-2024-26458",
                        "severity": "MEDIUM",
                        "pkgID": "libkrb5-3@1.20.1-2+deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-26458",
                        "title": "krb5: Memory leak at /krb5/src/lib/rpc/pmap_rmt.c",
                        "description": "Kerberos 5 (aka krb5) 1.21.2 contains a memory leak in /krb5/src/lib/rpc/pmap_rmt.c.",
                        "nvdV3Score": 0,
                        "publishedDate": "2024-02-29T01:44:18.78Z"
                    },
                    {
                        "registry": "CVE-2024-26461",
                        "severity": "MEDIUM",
                        "pkgID": "libkrb5-3@1.20.1-2+deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-26461",
                        "title": "krb5: Memory leak at /krb5/src/lib/gssapi/krb5/k5sealv3.c",
                        "description": "Kerberos 5 (aka krb5) 1.21.2 contains a memory leak vulnerability in /krb5/src/lib/gssapi/krb5/k5sealv3.c.",
                        "nvdV3Score": 0,
                        "publishedDate": "2024-02-29T01:44:18.82Z"
                    },
                    {
                        "registry": "CVE-2018-5709",
                        "severity": "LOW",
                        "pkgID": "libkrb5-3@1.20.1-2+deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2018-5709",
                        "title": "krb5: integer overflow in dbentry-\u003en_key_data in kadmin/dbutil/dump.c",
                        "description": "An issue was discovered in MIT Kerberos 5 (aka krb5) through 1.16. There is a variable \"dbentry-\u003en_key_data\" in kadmin/dbutil/dump.c that can store 16-bit data but unknowingly the developer has assigned a \"u4\" variable to it, which is for 32-bit data. An attacker can use this vulnerability to affect other artifacts of the database as we know that a Kerberos database dump file contains trusted data.",
                        "nvdV3Score": 7.5,
                        "publishedDate": "2018-01-16T09:29:00.5Z"
                    },
                    {
                        "registry": "CVE-2024-26462",
                        "severity": "HIGH",
                        "pkgID": "libkrb5support0@1.20.1-2+deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-26462",
                        "title": "krb5: Memory leak at /krb5/src/kdc/ndr.c",
                        "description": "Kerberos 5 (aka krb5) 1.21.2 contains a memory leak vulnerability in /krb5/src/kdc/ndr.c.",
                        "nvdV3Score": 0,
                        "publishedDate": "2024-02-29T01:44:18.857Z"
                    },
                    {
                        "registry": "CVE-2024-26458",
                        "severity": "MEDIUM",
                        "pkgID": "libkrb5support0@1.20.1-2+deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-26458",
                        "title": "krb5: Memory leak at /krb5/src/lib/rpc/pmap_rmt.c",
                        "description": "Kerberos 5 (aka krb5) 1.21.2 contains a memory leak in /krb5/src/lib/rpc/pmap_rmt.c.",
                        "nvdV3Score": 0,
                        "publishedDate": "2024-02-29T01:44:18.78Z"
                    },
                    {
                        "registry": "CVE-2024-26461",
                        "severity": "MEDIUM",
                        "pkgID": "libkrb5support0@1.20.1-2+deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-26461",
                        "title": "krb5: Memory leak at /krb5/src/lib/gssapi/krb5/k5sealv3.c",
                        "description": "Kerberos 5 (aka krb5) 1.21.2 contains a memory leak vulnerability in /krb5/src/lib/gssapi/krb5/k5sealv3.c.",
                        "nvdV3Score": 0,
                        "publishedDate": "2024-02-29T01:44:18.82Z"
                    },
                    {
                        "registry": "CVE-2018-5709",
                        "severity": "LOW",
                        "pkgID": "libkrb5support0@1.20.1-2+deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2018-5709",
                        "title": "krb5: integer overflow in dbentry-\u003en_key_data in kadmin/dbutil/dump.c",
                        "description": "An issue was discovered in MIT Kerberos 5 (aka krb5) through 1.16. There is a variable \"dbentry-\u003en_key_data\" in kadmin/dbutil/dump.c that can store 16-bit data but unknowingly the developer has assigned a \"u4\" variable to it, which is for 32-bit data. An attacker can use this vulnerability to affect other artifacts of the database as we know that a Kerberos database dump file contains trusted data.",
                        "nvdV3Score": 7.5,
                        "publishedDate": "2018-01-16T09:29:00.5Z"
                    },
                    {
                        "registry": "CVE-2023-2953",
                        "severity": "HIGH",
                        "pkgID": "libldap-2.5-0@2.5.13+dfsg-5",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-2953",
                        "title": "openldap: null pointer dereference in  ber_memalloc_x  function",
                        "description": "A vulnerability was found in openldap. This security flaw causes a null pointer dereference in ber_memalloc_x() function.",
                        "nvdV3Score": 7.5,
                        "publishedDate": "2023-05-30T22:15:10.613Z"
                    },
                    {
                        "registry": "CVE-2015-3276",
                        "severity": "LOW",
                        "pkgID": "libldap-2.5-0@2.5.13+dfsg-5",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2015-3276",
                        "title": "openldap: incorrect multi-keyword mode cipherstring parsing",
                        "description": "The nss_parse_ciphers function in libraries/libldap/tls_m.c in OpenLDAP does not properly parse OpenSSL-style multi-keyword mode cipher strings, which might cause a weaker than intended cipher to be used and allow remote attackers to have unspecified impact via unknown vectors.",
                        "nvdV3Score": 7.5,
                        "publishedDate": "2015-12-07T20:59:03.023Z"
                    },
                    {
                        "registry": "CVE-2017-14159",
                        "severity": "LOW",
                        "pkgID": "libldap-2.5-0@2.5.13+dfsg-5",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2017-14159",
                        "title": "openldap: Privilege escalation via PID file manipulation",
                        "description": "slapd in OpenLDAP 2.4.45 and earlier creates a PID file after dropping privileges to a non-root account, which might allow local users to kill arbitrary processes by leveraging access to this non-root account for PID file modification before a root script executes a \"kill `cat /pathname`\" command, as demonstrated by openldap-initscript.",
                        "nvdV3Score": 4.7,
                        "publishedDate": "2017-09-05T18:29:00.133Z"
                    },
                    {
                        "registry": "CVE-2017-17740",
                        "severity": "LOW",
                        "pkgID": "libldap-2.5-0@2.5.13+dfsg-5",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2017-17740",
                        "title": "openldap: contrib/slapd-modules/nops/nops.c attempts to free stack buffer allowing remote attackers to cause a denial of service",
                        "description": "contrib/slapd-modules/nops/nops.c in OpenLDAP through 2.4.45, when both the nops module and the memberof overlay are enabled, attempts to free a buffer that was allocated on the stack, which allows remote attackers to cause a denial of service (slapd crash) via a member MODDN operation.",
                        "nvdV3Score": 7.5,
                        "publishedDate": "2017-12-18T06:29:00.397Z"
                    },
                    {
                        "registry": "CVE-2020-15719",
                        "severity": "LOW",
                        "pkgID": "libldap-2.5-0@2.5.13+dfsg-5",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2020-15719",
                        "title": "openldap: Certificate validation incorrectly matches name against CN-ID",
                        "description": "libldap in certain third-party OpenLDAP packages has a certificate-validation flaw when the third-party package is asserting RFC6125 support. It considers CN even when there is a non-matching subjectAltName (SAN). This is fixed in, for example, openldap-2.4.46-10.el8 in Red Hat Enterprise Linux.",
                        "nvdV3Score": 4.2,
                        "publishedDate": "2020-07-14T14:15:17.667Z"
                    },
                    {
                        "registry": "CVE-2022-0563",
                        "severity": "LOW",
                        "pkgID": "libmount1@2.38.1-5+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2022-0563",
                        "title": "util-linux: partial disclosure of arbitrary files in chfn and chsh when compiled with libreadline",
                        "description": "A flaw was found in the util-linux chfn and chsh utilities when compiled with Readline support. The Readline library uses an \"INPUTRC\" environment variable to get a path to the library config file. When the library cannot parse the specified file, it prints an error message containing data from the file. This flaw allows an unprivileged user to read root-owned files, potentially leading to privilege escalation. This flaw affects util-linux versions prior to 2.37.4.",
                        "nvdV3Score": 5.5,
                        "publishedDate": "2022-02-21T19:15:08.393Z"
                    },
                    {
                        "registry": "CVE-2024-28182",
                        "severity": "MEDIUM",
                        "pkgID": "libnghttp2-14@1.52.0-1+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-28182",
                        "title": "nghttp2: CONTINUATION frames DoS",
                        "description": "nghttp2 is an implementation of the Hypertext Transfer Protocol version 2 in C. The nghttp2 library prior to version 1.61.0 keeps reading the unbounded number of HTTP/2 CONTINUATION frames even after a stream is reset to keep HPACK context in sync.  This causes excessive CPU usage to decode HPACK stream. nghttp2 v1.61.0 mitigates this vulnerability by limiting the number of CONTINUATION frames it accepts per stream. There is no workaround for this vulnerability.",
                        "nvdV3Score": 0,
                        "publishedDate": "2024-04-04T15:15:38.427Z"
                    },
                    {
                        "registry": "CVE-2024-22365",
                        "severity": "MEDIUM",
                        "pkgID": "libpam-modules@1.5.2-6+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-22365",
                        "title": "pam: allowing unprivileged user to block another user namespace",
                        "description": "linux-pam (aka Linux PAM) before 1.6.0 allows attackers to cause a denial of service (blocked login process) via mkfifo because the openat call (for protect_dir) lacks O_DIRECTORY.",
                        "nvdV3Score": 5.5,
                        "publishedDate": "2024-02-06T08:15:52.203Z"
                    },
                    {
                        "registry": "CVE-2024-22365",
                        "severity": "MEDIUM",
                        "pkgID": "libpam-modules-bin@1.5.2-6+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-22365",
                        "title": "pam: allowing unprivileged user to block another user namespace",
                        "description": "linux-pam (aka Linux PAM) before 1.6.0 allows attackers to cause a denial of service (blocked login process) via mkfifo because the openat call (for protect_dir) lacks O_DIRECTORY.",
                        "nvdV3Score": 5.5,
                        "publishedDate": "2024-02-06T08:15:52.203Z"
                    },
                    {
                        "registry": "CVE-2024-22365",
                        "severity": "MEDIUM",
                        "pkgID": "libpam-runtime@1.5.2-6+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-22365",
                        "title": "pam: allowing unprivileged user to block another user namespace",
                        "description": "linux-pam (aka Linux PAM) before 1.6.0 allows attackers to cause a denial of service (blocked login process) via mkfifo because the openat call (for protect_dir) lacks O_DIRECTORY.",
                        "nvdV3Score": 5.5,
                        "publishedDate": "2024-02-06T08:15:52.203Z"
                    },
                    {
                        "registry": "CVE-2024-22365",
                        "severity": "MEDIUM",
                        "pkgID": "libpam0g@1.5.2-6+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-22365",
                        "title": "pam: allowing unprivileged user to block another user namespace",
                        "description": "linux-pam (aka Linux PAM) before 1.6.0 allows attackers to cause a denial of service (blocked login process) via mkfifo because the openat call (for protect_dir) lacks O_DIRECTORY.",
                        "nvdV3Score": 5.5,
                        "publishedDate": "2024-02-06T08:15:52.203Z"
                    },
                    {
                        "registry": "CVE-2021-4214",
                        "severity": "LOW",
                        "pkgID": "libpng16-16@1.6.39-2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2021-4214",
                        "title": "libpng: hardcoded value leads to heap-overflow",
                        "description": "A heap overflow flaw was found in libpngs' pngimage.c program. This flaw allows an attacker with local network access to pass a specially crafted PNG file to the pngimage utility, causing an application to crash, leading to a denial of service.",
                        "nvdV3Score": 5.5,
                        "publishedDate": "2022-08-24T16:15:10.037Z"
                    },
                    {
                        "registry": "CVE-2022-0563",
                        "severity": "LOW",
                        "pkgID": "libsmartcols1@2.38.1-5+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2022-0563",
                        "title": "util-linux: partial disclosure of arbitrary files in chfn and chsh when compiled with libreadline",
                        "description": "A flaw was found in the util-linux chfn and chsh utilities when compiled with Readline support. The Readline library uses an \"INPUTRC\" environment variable to get a path to the library config file. When the library cannot parse the specified file, it prints an error message containing data from the file. This flaw allows an unprivileged user to read root-owned files, potentially leading to privilege escalation. This flaw affects util-linux versions prior to 2.37.4.",
                        "nvdV3Score": 5.5,
                        "publishedDate": "2022-02-21T19:15:08.393Z"
                    },
                    {
                        "registry": "CVE-2024-5535",
                        "severity": "MEDIUM",
                        "pkgID": "libssl3@3.0.14-1~deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-5535",
                        "title": "openssl: SSL_select_next_proto buffer overread",
                        "description": "Issue summary: Calling the OpenSSL API function SSL_select_next_proto with an\nempty supported client protocols buffer may cause a crash or memory contents to\nbe sent to the peer.\n\nImpact summary: A buffer overread can have a range of potential consequences\nsuch as unexpected application beahviour or a crash. In particular this issue\ncould result in up to 255 bytes of arbitrary private data from memory being sent\nto the peer leading to a loss of confidentiality. However, only applications\nthat directly call the SSL_select_next_proto function with a 0 length list of\nsupported client protocols are affected by this issue. This would normally never\nbe a valid scenario and is typically not under attacker control but may occur by\naccident in the case of a configuration or programming error in the calling\napplication.\n\nThe OpenSSL API function SSL_select_next_proto is typically used by TLS\napplications that support ALPN (Application Layer Protocol Negotiation) or NPN\n(Next Protocol Negotiation). NPN is older, was never standardised and\nis deprecated in favour of ALPN. We believe that ALPN is significantly more\nwidely deployed than NPN. The SSL_select_next_proto function accepts a list of\nprotocols from the server and a list of protocols from the client and returns\nthe first protocol that appears in the server list that also appears in the\nclient list. In the case of no overlap between the two lists it returns the\nfirst item in the client list. In either case it will signal whether an overlap\nbetween the two lists was found. In the case where SSL_select_next_proto is\ncalled with a zero length client list it fails to notice this condition and\nreturns the memory immediately following the client list pointer (and reports\nthat there was no overlap in the lists).\n\nThis function is typically called from a server side application callback for\nALPN or a client side application callback for NPN. In the case of ALPN the list\nof protocols supplied by the client is guaranteed by libssl to never be zero in\nlength. The list of server protocols comes from the application and should never\nnormally be expected to be of zero length. In this case if the\nSSL_select_next_proto function has been called as expected (with the list\nsupplied by the client passed in the client/client_len parameters), then the\napplication will not be vulnerable to this issue. If the application has\naccidentally been configured with a zero length server list, and has\naccidentally passed that zero length server list in the client/client_len\nparameters, and has additionally failed to correctly handle a \"no overlap\"\nresponse (which would normally result in a handshake failure in ALPN) then it\nwill be vulnerable to this problem.\n\nIn the case of NPN, the protocol permits the client to opportunistically select\na protocol when there is no overlap. OpenSSL returns the first client protocol\nin the no overlap case in support of this. The list of client protocols comes\nfrom the application and should never normally be expected to be of zero length.\nHowever if the SSL_select_next_proto function is accidentally called with a\nclient_len of 0 then an invalid memory pointer will be returned instead. If the\napplication uses this output as the opportunistic protocol then the loss of\nconfidentiality will occur.\n\nThis issue has been assessed as Low severity because applications are most\nlikely to be vulnerable if they are using NPN instead of ALPN - but NPN is not\nwidely used. It also requires an application configuration or programming error.\nFinally, this issue would not typically be under attacker control making active\nexploitation unlikely.\n\nThe FIPS modules in 3.3, 3.2, 3.1 and 3.0 are not affected by this issue.\n\nDue to the low severity of this issue we are not issuing new releases of\nOpenSSL at this time. The fix will be included in the next releases when they\nbecome available.",
                        "nvdV3Score": 0,
                        "publishedDate": "2024-06-27T11:15:24.447Z"
                    },
                    {
                        "registry": "CVE-2023-4039",
                        "severity": "MEDIUM",
                        "pkgID": "libstdc++6@12.2.0-14",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-4039",
                        "title": "gcc: -fstack-protector fails to guard dynamic stack allocations on ARM64",
                        "description": "\n\n**DISPUTED**A failure in the -fstack-protector feature in GCC-based toolchains \nthat target AArch64 allows an attacker to exploit an existing buffer \noverflow in dynamically-sized local variables in your application \nwithout this being detected. This stack-protector failure only applies \nto C99-style dynamically-sized local variables or those created using \nalloca(). The stack-protector operates as intended for statically-sized \nlocal variables.\n\nThe default behavior when the stack-protector \ndetects an overflow is to terminate your application, resulting in \ncontrolled loss of availability. An attacker who can exploit a buffer \noverflow without triggering the stack-protector might be able to change \nprogram flow control to cause an uncontrolled loss of availability or to\n go further and affect confidentiality or integrity. NOTE: The GCC project argues that this is a missed hardening bug and not a vulnerability by itself.\n\n\n\n\n\n",
                        "nvdV3Score": 4.8,
                        "publishedDate": "2023-09-13T09:15:15.69Z"
                    },
                    {
                        "registry": "CVE-2022-27943",
                        "severity": "LOW",
                        "pkgID": "libstdc++6@12.2.0-14",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2022-27943",
                        "title": "binutils: libiberty/rust-demangle.c in GNU GCC 11.2 allows stack exhaustion in demangle_const",
                        "description": "libiberty/rust-demangle.c in GNU GCC 11.2 allows stack consumption in demangle_const, as demonstrated by nm-new.",
                        "nvdV3Score": 5.5,
                        "publishedDate": "2022-03-26T13:15:07.9Z"
                    },
                    {
                        "registry": "CVE-2013-4392",
                        "severity": "LOW",
                        "pkgID": "libsystemd0@252.30-1~deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2013-4392",
                        "title": "systemd: TOCTOU race condition when updating file permissions and SELinux security contexts",
                        "description": "systemd, when updating file permissions, allows local users to change the permissions and SELinux security contexts for arbitrary files via a symlink attack on unspecified files.",
                        "nvdV3Score": 0,
                        "publishedDate": "2013-10-28T22:55:03.773Z"
                    },
                    {
                        "registry": "CVE-2023-31437",
                        "severity": "LOW",
                        "pkgID": "libsystemd0@252.30-1~deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-31437",
                        "title": "An issue was discovered in systemd 253. An attacker can modify a seale ...",
                        "description": "An issue was discovered in systemd 253. An attacker can modify a sealed log file such that, in some views, not all existing and sealed log messages are displayed. NOTE: the vendor reportedly sent \"a reply denying that any of the finding was a security vulnerability.\"",
                        "nvdV3Score": 5.3,
                        "publishedDate": "2023-06-13T17:15:14.657Z"
                    },
                    {
                        "registry": "CVE-2023-31438",
                        "severity": "LOW",
                        "pkgID": "libsystemd0@252.30-1~deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-31438",
                        "title": "An issue was discovered in systemd 253. An attacker can truncate a sea ...",
                        "description": "An issue was discovered in systemd 253. An attacker can truncate a sealed log file and then resume log sealing such that checking the integrity shows no error, despite modifications. NOTE: the vendor reportedly sent \"a reply denying that any of the finding was a security vulnerability.\"",
                        "nvdV3Score": 5.3,
                        "publishedDate": "2023-06-13T17:15:14.707Z"
                    },
                    {
                        "registry": "CVE-2023-31439",
                        "severity": "LOW",
                        "pkgID": "libsystemd0@252.30-1~deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-31439",
                        "title": "An issue was discovered in systemd 253. An attacker can modify the con ...",
                        "description": "An issue was discovered in systemd 253. An attacker can modify the contents of past events in a sealed log file and then adjust the file such that checking the integrity shows no error, despite modifications. NOTE: the vendor reportedly sent \"a reply denying that any of the finding was a security vulnerability.\"",
                        "nvdV3Score": 5.3,
                        "publishedDate": "2023-06-13T17:15:14.753Z"
                    },
                    {
                        "registry": "CVE-2023-52355",
                        "severity": "HIGH",
                        "pkgID": "libtiff6@4.5.0-6+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-52355",
                        "title": "libtiff: TIFFRasterScanlineSize64 produce too-big size and could cause OOM",
                        "description": "An out-of-memory flaw was found in libtiff that could be triggered by passing a crafted tiff file to the TIFFRasterScanlineSize64() API. This flaw allows a remote attacker to cause a denial of service via a crafted input with a size smaller than 379 KB.",
                        "nvdV3Score": 7.5,
                        "publishedDate": "2024-01-25T20:15:38.353Z"
                    },
                    {
                        "registry": "CVE-2023-52356",
                        "severity": "HIGH",
                        "pkgID": "libtiff6@4.5.0-6+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-52356",
                        "title": "libtiff: Segment fault in libtiff  in TIFFReadRGBATileExt() leading to denial of service",
                        "description": "A segment fault (SEGV) flaw was found in libtiff that could be triggered by passing a crafted tiff file to the TIFFReadRGBATileExt() API. This flaw allows a remote attacker to cause a heap-buffer overflow, leading to a denial of service.",
                        "nvdV3Score": 7.5,
                        "publishedDate": "2024-01-25T20:15:39.063Z"
                    },
                    {
                        "registry": "CVE-2024-7006",
                        "severity": "HIGH",
                        "pkgID": "libtiff6@4.5.0-6+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-7006",
                        "title": "libtiff: NULL pointer dereference in tif_dirinfo.c",
                        "description": "A null pointer dereference flaw was found in Libtiff via `tif_dirinfo.c`. This issue may allow an attacker to trigger memory allocation failures through certain means, such as restricting the heap space size or injecting faults, causing a segmentation fault. This can cause an application crash, eventually leading to a denial of service.",
                        "nvdV3Score": 7.5,
                        "publishedDate": "2024-08-12T13:38:40.577Z"
                    },
                    {
                        "registry": "CVE-2023-25433",
                        "severity": "MEDIUM",
                        "pkgID": "libtiff6@4.5.0-6+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-25433",
                        "title": "libtiff: Buffer Overflow via /libtiff/tools/tiffcrop.c",
                        "description": "libtiff 4.5.0 is vulnerable to Buffer Overflow via /libtiff/tools/tiffcrop.c:8499. Incorrect updating of buffer size after rotateImage() in tiffcrop cause heap-buffer-overflow and SEGV.",
                        "nvdV3Score": 5.5,
                        "publishedDate": "2023-06-29T20:15:09.83Z"
                    },
                    {
                        "registry": "CVE-2023-26965",
                        "severity": "MEDIUM",
                        "pkgID": "libtiff6@4.5.0-6+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-26965",
                        "title": "libtiff: heap-based use after free via a crafted TIFF image in loadImage() in tiffcrop.c",
                        "description": "loadImage() in tools/tiffcrop.c in LibTIFF through 4.5.0 has a heap-based use after free via a crafted TIFF image.",
                        "nvdV3Score": 5.5,
                        "publishedDate": "2023-06-14T21:15:09.483Z"
                    },
                    {
                        "registry": "CVE-2023-26966",
                        "severity": "MEDIUM",
                        "pkgID": "libtiff6@4.5.0-6+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-26966",
                        "title": "libtiff: Buffer Overflow in uv_encode()",
                        "description": "libtiff 4.5.0 is vulnerable to Buffer Overflow in uv_encode() when libtiff reads a corrupted little-endian TIFF file and specifies the output to be big-endian.",
                        "nvdV3Score": 5.5,
                        "publishedDate": "2023-06-29T20:15:09.873Z"
                    },
                    {
                        "registry": "CVE-2023-2908",
                        "severity": "MEDIUM",
                        "pkgID": "libtiff6@4.5.0-6+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-2908",
                        "title": "libtiff: null pointer dereference in tif_dir.c",
                        "description": "A null pointer dereference issue was found in Libtiff's tif_dir.c file. This issue may allow an attacker to pass a crafted TIFF image file to the tiffcp utility which triggers a runtime error that causes undefined behavior. This will result in an application crash, eventually leading to a denial of service.",
                        "nvdV3Score": 5.5,
                        "publishedDate": "2023-06-30T22:15:10.017Z"
                    },
                    {
                        "registry": "CVE-2023-3618",
                        "severity": "MEDIUM",
                        "pkgID": "libtiff6@4.5.0-6+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-3618",
                        "title": "libtiff: segmentation fault in Fax3Encode in libtiff/tif_fax3.c",
                        "description": "A flaw was found in libtiff. A specially crafted tiff file can lead to a segmentation fault due to a buffer overflow in the Fax3Encode function in libtiff/tif_fax3.c, resulting in a denial of service.",
                        "nvdV3Score": 6.5,
                        "publishedDate": "2023-07-12T15:15:09.06Z"
                    },
                    {
                        "registry": "CVE-2023-6277",
                        "severity": "MEDIUM",
                        "pkgID": "libtiff6@4.5.0-6+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-6277",
                        "title": "libtiff: Out-of-memory in TIFFOpen via a craft file",
                        "description": "An out-of-memory flaw was found in libtiff. Passing a crafted tiff file to TIFFOpen() API may allow a remote attacker to cause a denial of service via a craft input with size smaller than 379 KB.",
                        "nvdV3Score": 6.5,
                        "publishedDate": "2023-11-24T19:15:07.643Z"
                    },
                    {
                        "registry": "CVE-2017-16232",
                        "severity": "LOW",
                        "pkgID": "libtiff6@4.5.0-6+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2017-16232",
                        "title": "libtiff: Memory leaks in tif_open.c, tif_lzw.c, and tif_aux.c",
                        "description": "LibTIFF 4.0.8 has multiple memory leak vulnerabilities, which allow attackers to cause a denial of service (memory consumption), as demonstrated by tif_open.c, tif_lzw.c, and tif_aux.c. NOTE: Third parties were unable to reproduce the issue",
                        "nvdV3Score": 7.5,
                        "publishedDate": "2019-03-21T15:59:56.53Z"
                    },
                    {
                        "registry": "CVE-2017-17973",
                        "severity": "LOW",
                        "pkgID": "libtiff6@4.5.0-6+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2017-17973",
                        "title": "libtiff: heap-based use after free in tiff2pdf.c:t2p_writeproc",
                        "description": "In LibTIFF 4.0.8, there is a heap-based use-after-free in the t2p_writeproc function in tiff2pdf.c. NOTE: there is a third-party report of inability to reproduce this issue",
                        "nvdV3Score": 8.8,
                        "publishedDate": "2017-12-29T21:29:00.19Z"
                    },
                    {
                        "registry": "CVE-2017-5563",
                        "severity": "LOW",
                        "pkgID": "libtiff6@4.5.0-6+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2017-5563",
                        "title": "libtiff: Heap-buffer overflow in LZWEncode tif_lzw.c",
                        "description": "LibTIFF version 4.0.7 is vulnerable to a heap-based buffer over-read in tif_lzw.c resulting in DoS or code execution via a crafted bmp image to tools/bmp2tiff.",
                        "nvdV3Score": 8.8,
                        "publishedDate": "2017-01-23T07:59:00.69Z"
                    },
                    {
                        "registry": "CVE-2017-9117",
                        "severity": "LOW",
                        "pkgID": "libtiff6@4.5.0-6+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2017-9117",
                        "title": "libtiff: Heap-based buffer over-read in bmp2tiff",
                        "description": "In LibTIFF 4.0.7, the program processes BMP images without verifying that biWidth and biHeight in the bitmap-information header match the actual input, leading to a heap-based buffer over-read in bmp2tiff.",
                        "nvdV3Score": 9.8,
                        "publishedDate": "2017-05-21T19:29:00.187Z"
                    },
                    {
                        "registry": "CVE-2018-10126",
                        "severity": "LOW",
                        "pkgID": "libtiff6@4.5.0-6+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2018-10126",
                        "title": "libtiff: NULL pointer dereference in the jpeg_fdct_16x16 function in jfdctint.c",
                        "description": "ijg-libjpeg before 9d, as used in tiff2pdf (from LibTIFF) and other products, does not check for a NULL pointer at a certain place in jpeg_fdct_16x16 in jfdctint.c.",
                        "nvdV3Score": 6.5,
                        "publishedDate": "2018-04-21T21:29:00.29Z"
                    },
                    {
                        "registry": "CVE-2022-1210",
                        "severity": "LOW",
                        "pkgID": "libtiff6@4.5.0-6+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2022-1210",
                        "title": "tiff: Malicious file leads to a denial of service in TIFF File Handler",
                        "description": "A vulnerability classified as problematic was found in LibTIFF 4.3.0. Affected by this vulnerability is the TIFF File Handler of tiff2ps. Opening a malicious file leads to a denial of service. The attack can be launched remotely but requires user interaction. The exploit has been disclosed to the public and may be used.",
                        "nvdV3Score": 6.5,
                        "publishedDate": "2022-04-03T09:15:09.033Z"
                    },
                    {
                        "registry": "CVE-2023-1916",
                        "severity": "LOW",
                        "pkgID": "libtiff6@4.5.0-6+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-1916",
                        "title": "libtiff: out-of-bounds read in extractImageSection() in tools/tiffcrop.c",
                        "description": "A flaw was found in tiffcrop, a program distributed by the libtiff package. A specially crafted tiff file can lead to an out-of-bounds read in the extractImageSection function in tools/tiffcrop.c, resulting in a denial of service and limited information disclosure. This issue affects libtiff versions 4.x.",
                        "nvdV3Score": 6.1,
                        "publishedDate": "2023-04-10T22:15:09.223Z"
                    },
                    {
                        "registry": "CVE-2023-3164",
                        "severity": "LOW",
                        "pkgID": "libtiff6@4.5.0-6+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-3164",
                        "title": "libtiff: heap-buffer-overflow in extractImageSection()",
                        "description": "A heap-buffer-overflow vulnerability was found in LibTIFF, in extractImageSection() at tools/tiffcrop.c:7916 and tools/tiffcrop.c:7801. This flaw allows attackers to cause a denial of service via a crafted tiff file.",
                        "nvdV3Score": 5.5,
                        "publishedDate": "2023-11-02T12:15:09.543Z"
                    },
                    {
                        "registry": "CVE-2023-6228",
                        "severity": "LOW",
                        "pkgID": "libtiff6@4.5.0-6+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-6228",
                        "title": "libtiff: heap-based buffer overflow in cpStripToTile() in tools/tiffcp.c",
                        "description": "An issue was found in the tiffcp utility distributed by the libtiff package where a crafted TIFF file on processing may cause a heap-based buffer overflow leads to an application crash.",
                        "nvdV3Score": 5.5,
                        "publishedDate": "2023-12-18T14:15:11.84Z"
                    },
                    {
                        "registry": "CVE-2023-50495",
                        "severity": "MEDIUM",
                        "pkgID": "libtinfo6@6.4-4",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-50495",
                        "title": "ncurses: segmentation fault via _nc_wrap_entry()",
                        "description": "NCurse v6.4-20230418 was discovered to contain a segmentation fault via the component _nc_wrap_entry().",
                        "nvdV3Score": 6.5,
                        "publishedDate": "2023-12-12T15:15:07.867Z"
                    },
                    {
                        "registry": "CVE-2023-45918",
                        "severity": "LOW",
                        "pkgID": "libtinfo6@6.4-4",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-45918",
                        "title": "ncurses: NULL pointer dereference in tgetstr in tinfo/lib_termcap.c",
                        "description": "ncurses 6.4-20230610 has a NULL pointer dereference in tgetstr in tinfo/lib_termcap.c.",
                        "nvdV3Score": 0,
                        "publishedDate": "2024-02-16T22:15:07.88Z"
                    },
                    {
                        "registry": "CVE-2013-4392",
                        "severity": "LOW",
                        "pkgID": "libudev1@252.30-1~deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2013-4392",
                        "title": "systemd: TOCTOU race condition when updating file permissions and SELinux security contexts",
                        "description": "systemd, when updating file permissions, allows local users to change the permissions and SELinux security contexts for arbitrary files via a symlink attack on unspecified files.",
                        "nvdV3Score": 0,
                        "publishedDate": "2013-10-28T22:55:03.773Z"
                    },
                    {
                        "registry": "CVE-2023-31437",
                        "severity": "LOW",
                        "pkgID": "libudev1@252.30-1~deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-31437",
                        "title": "An issue was discovered in systemd 253. An attacker can modify a seale ...",
                        "description": "An issue was discovered in systemd 253. An attacker can modify a sealed log file such that, in some views, not all existing and sealed log messages are displayed. NOTE: the vendor reportedly sent \"a reply denying that any of the finding was a security vulnerability.\"",
                        "nvdV3Score": 5.3,
                        "publishedDate": "2023-06-13T17:15:14.657Z"
                    },
                    {
                        "registry": "CVE-2023-31438",
                        "severity": "LOW",
                        "pkgID": "libudev1@252.30-1~deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-31438",
                        "title": "An issue was discovered in systemd 253. An attacker can truncate a sea ...",
                        "description": "An issue was discovered in systemd 253. An attacker can truncate a sealed log file and then resume log sealing such that checking the integrity shows no error, despite modifications. NOTE: the vendor reportedly sent \"a reply denying that any of the finding was a security vulnerability.\"",
                        "nvdV3Score": 5.3,
                        "publishedDate": "2023-06-13T17:15:14.707Z"
                    },
                    {
                        "registry": "CVE-2023-31439",
                        "severity": "LOW",
                        "pkgID": "libudev1@252.30-1~deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-31439",
                        "title": "An issue was discovered in systemd 253. An attacker can modify the con ...",
                        "description": "An issue was discovered in systemd 253. An attacker can modify the contents of past events in a sealed log file and then adjust the file such that checking the integrity shows no error, despite modifications. NOTE: the vendor reportedly sent \"a reply denying that any of the finding was a security vulnerability.\"",
                        "nvdV3Score": 5.3,
                        "publishedDate": "2023-06-13T17:15:14.753Z"
                    },
                    {
                        "registry": "CVE-2022-0563",
                        "severity": "LOW",
                        "pkgID": "libuuid1@2.38.1-5+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2022-0563",
                        "title": "util-linux: partial disclosure of arbitrary files in chfn and chsh when compiled with libreadline",
                        "description": "A flaw was found in the util-linux chfn and chsh utilities when compiled with Readline support. The Readline library uses an \"INPUTRC\" environment variable to get a path to the library config file. When the library cannot parse the specified file, it prints an error message containing data from the file. This flaw allows an unprivileged user to read root-owned files, potentially leading to privilege escalation. This flaw affects util-linux versions prior to 2.37.4.",
                        "nvdV3Score": 5.5,
                        "publishedDate": "2022-02-21T19:15:08.393Z"
                    },
                    {
                        "registry": "CVE-2024-25062",
                        "severity": "HIGH",
                        "pkgID": "libxml2@2.9.14+dfsg-1.3~deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-25062",
                        "title": "libxml2: use-after-free in XMLReader",
                        "description": "An issue was discovered in libxml2 before 2.11.7 and 2.12.x before 2.12.5. When using the XML Reader interface with DTD validation and XInclude expansion enabled, processing crafted XML documents can lead to an xmlValidatePopElement use-after-free.",
                        "nvdV3Score": 7.5,
                        "publishedDate": "2024-02-04T16:15:45.12Z"
                    },
                    {
                        "registry": "CVE-2023-39615",
                        "severity": "MEDIUM",
                        "pkgID": "libxml2@2.9.14+dfsg-1.3~deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-39615",
                        "title": "libxml2: crafted xml can cause global buffer overflow",
                        "description": "Xmlsoft Libxml2 v2.11.0 was discovered to contain an out-of-bounds read via the xmlSAX2StartElement() function at /libxml2/SAX2.c. This vulnerability allows attackers to cause a Denial of Service (DoS) via supplying a crafted XML file. NOTE: the vendor's position is that the product does not support the legacy SAX1 interface with custom callbacks; there is a crash even without crafted input.",
                        "nvdV3Score": 6.5,
                        "publishedDate": "2023-08-29T17:15:12.527Z"
                    },
                    {
                        "registry": "CVE-2023-45322",
                        "severity": "MEDIUM",
                        "pkgID": "libxml2@2.9.14+dfsg-1.3~deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-45322",
                        "title": "libxml2: use-after-free in xmlUnlinkNode() in tree.c",
                        "description": "libxml2 through 2.11.5 has a use-after-free that can only occur after a certain memory allocation fails. This occurs in xmlUnlinkNode in tree.c. NOTE: the vendor's position is \"I don't think these issues are critical enough to warrant a CVE ID ... because an attacker typically can't control when memory allocations fail.\"",
                        "nvdV3Score": 6.5,
                        "publishedDate": "2023-10-06T22:15:11.66Z"
                    },
                    {
                        "registry": "CVE-2024-34459",
                        "severity": "LOW",
                        "pkgID": "libxml2@2.9.14+dfsg-1.3~deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-34459",
                        "title": "libxml2: buffer over-read in xmlHTMLPrintFileContext in xmllint.c",
                        "description": "An issue was discovered in xmllint (from libxml2) before 2.11.8 and 2.12.x before 2.12.7. Formatting error messages with xmllint --htmlout can result in a buffer over-read in xmlHTMLPrintFileContext in xmllint.c.",
                        "nvdV3Score": 0,
                        "publishedDate": "2024-05-14T15:39:11.917Z"
                    },
                    {
                        "registry": "CVE-2015-9019",
                        "severity": "LOW",
                        "pkgID": "libxslt1.1@1.1.35-1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2015-9019",
                        "title": "libxslt: math.random() in xslt uses unseeded randomness",
                        "description": "In libxslt 1.1.29 and earlier, the EXSLT math.random function was not initialized with a random seed during startup, which could cause usage of this function to produce predictable outputs.",
                        "nvdV3Score": 5.3,
                        "publishedDate": "2017-04-05T21:59:00.147Z"
                    },
                    {
                        "registry": "CVE-2023-4641",
                        "severity": "MEDIUM",
                        "pkgID": "login@1:4.13+dfsg1-1+b1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-4641",
                        "title": "shadow-utils: possible password leak during passwd(1) change",
                        "description": "A flaw was found in shadow-utils. When asking for a new password, shadow-utils asks the password twice. If the password fails on the second attempt, shadow-utils fails in cleaning the buffer used to store the first entry. This may allow an attacker with enough access to retrieve the password from the memory.",
                        "nvdV3Score": 5.5,
                        "publishedDate": "2023-12-27T16:15:13.363Z"
                    },
                    {
                        "registry": "CVE-2007-5686",
                        "severity": "LOW",
                        "pkgID": "login@1:4.13+dfsg1-1+b1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2007-5686",
                        "title": "initscripts in rPath Linux 1 sets insecure permissions for the /var/lo ...",
                        "description": "initscripts in rPath Linux 1 sets insecure permissions for the /var/log/btmp file, which allows local users to obtain sensitive information regarding authentication attempts.  NOTE: because sshd detects the insecure permissions and does not log certain events, this also prevents sshd from logging failed authentication attempts by remote attackers.",
                        "nvdV3Score": 0,
                        "publishedDate": "2007-10-28T17:08:00Z"
                    },
                    {
                        "registry": "CVE-2019-19882",
                        "severity": "LOW",
                        "pkgID": "login@1:4.13+dfsg1-1+b1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2019-19882",
                        "title": "shadow-utils: local users can obtain root access because setuid programs are misconfigured",
                        "description": "shadow 4.8, in certain circumstances affecting at least Gentoo, Arch Linux, and Void Linux, allows local users to obtain root access because setuid programs are misconfigured. Specifically, this affects shadow 4.8 when compiled using --with-libpam but without explicitly passing --disable-account-tools-setuid, and without a PAM configuration suitable for use with setuid account management tools. This combination leads to account management tools (groupadd, groupdel, groupmod, useradd, userdel, usermod) that can easily be used by unprivileged local users to escalate privileges to root in multiple ways. This issue became much more relevant in approximately December 2019 when an unrelated bug was fixed (i.e., the chmod calls to suidusbins were fixed in the upstream Makefile which is now included in the release version 4.8).",
                        "nvdV3Score": 7.8,
                        "publishedDate": "2019-12-18T16:15:26.963Z"
                    },
                    {
                        "registry": "CVE-2023-29383",
                        "severity": "LOW",
                        "pkgID": "login@1:4.13+dfsg1-1+b1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-29383",
                        "title": "shadow: Improper input validation in shadow-utils package utility chfn",
                        "description": "In Shadow 4.13, it is possible to inject control characters into fields provided to the SUID program chfn (change finger). Although it is not possible to exploit this directly (e.g., adding a new user fails because \\n is in the block list), it is possible to misrepresent the /etc/passwd file when viewed. Use of \\r manipulations and Unicode characters to work around blocking of the : character make it possible to give the impression that a new user has been added. In other words, an adversary may be able to convince a system administrator to take the system offline (an indirect, social-engineered denial of service) by demonstrating that \"cat /etc/passwd\" shows a rogue user account.",
                        "nvdV3Score": 3.3,
                        "publishedDate": "2023-04-14T22:15:07.68Z"
                    },
                    {
                        "registry": "TEMP-0628843-DBAD28",
                        "severity": "LOW",
                        "pkgID": "login@1:4.13+dfsg1-1+b1",
                        "primaryURL": "https://security-tracker.debian.org/tracker/TEMP-0628843-DBAD28",
                        "title": "[more related to CVE-2005-4890]",
                        "description": "",
                        "nvdV3Score": 0,
                        "publishedDate": null
                    },
                    {
                        "registry": "CVE-2022-0563",
                        "severity": "LOW",
                        "pkgID": "mount@2.38.1-5+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2022-0563",
                        "title": "util-linux: partial disclosure of arbitrary files in chfn and chsh when compiled with libreadline",
                        "description": "A flaw was found in the util-linux chfn and chsh utilities when compiled with Readline support. The Readline library uses an \"INPUTRC\" environment variable to get a path to the library config file. When the library cannot parse the specified file, it prints an error message containing data from the file. This flaw allows an unprivileged user to read root-owned files, potentially leading to privilege escalation. This flaw affects util-linux versions prior to 2.37.4.",
                        "nvdV3Score": 5.5,
                        "publishedDate": "2022-02-21T19:15:08.393Z"
                    },
                    {
                        "registry": "CVE-2023-50495",
                        "severity": "MEDIUM",
                        "pkgID": "ncurses-base@6.4-4",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-50495",
                        "title": "ncurses: segmentation fault via _nc_wrap_entry()",
                        "description": "NCurse v6.4-20230418 was discovered to contain a segmentation fault via the component _nc_wrap_entry().",
                        "nvdV3Score": 6.5,
                        "publishedDate": "2023-12-12T15:15:07.867Z"
                    },
                    {
                        "registry": "CVE-2023-45918",
                        "severity": "LOW",
                        "pkgID": "ncurses-base@6.4-4",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-45918",
                        "title": "ncurses: NULL pointer dereference in tgetstr in tinfo/lib_termcap.c",
                        "description": "ncurses 6.4-20230610 has a NULL pointer dereference in tgetstr in tinfo/lib_termcap.c.",
                        "nvdV3Score": 0,
                        "publishedDate": "2024-02-16T22:15:07.88Z"
                    },
                    {
                        "registry": "CVE-2023-50495",
                        "severity": "MEDIUM",
                        "pkgID": "ncurses-bin@6.4-4",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-50495",
                        "title": "ncurses: segmentation fault via _nc_wrap_entry()",
                        "description": "NCurse v6.4-20230418 was discovered to contain a segmentation fault via the component _nc_wrap_entry().",
                        "nvdV3Score": 6.5,
                        "publishedDate": "2023-12-12T15:15:07.867Z"
                    },
                    {
                        "registry": "CVE-2023-45918",
                        "severity": "LOW",
                        "pkgID": "ncurses-bin@6.4-4",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-45918",
                        "title": "ncurses: NULL pointer dereference in tgetstr in tinfo/lib_termcap.c",
                        "description": "ncurses 6.4-20230610 has a NULL pointer dereference in tgetstr in tinfo/lib_termcap.c.",
                        "nvdV3Score": 0,
                        "publishedDate": "2024-02-16T22:15:07.88Z"
                    },
                    {
                        "registry": "CVE-2024-7347",
                        "severity": "MEDIUM",
                        "pkgID": "nginx@1.27.1-1~bookworm",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-7347",
                        "title": "nginx: specially crafted MP4 file may cause denial of service",
                        "description": "NGINX Open Source and NGINX Plus have a vulnerability in the ngx_http_mp4_module, which might allow an attacker to over-read NGINX worker memory resulting in its termination, using a specially crafted mp4 file. The issue only affects NGINX if it is built with the ngx_http_mp4_module and the mp4 directive is used in the configuration file. Additionally, the attack is possible only if an attacker can trigger the processing of a specially crafted mp4 file with the ngx_http_mp4_module.  Note: Software versions which have reached End of Technical Support (EoTS) are not evaluated.",
                        "nvdV3Score": 4.7,
                        "publishedDate": "2024-08-14T15:15:31.87Z"
                    },
                    {
                        "registry": "CVE-2009-4487",
                        "severity": "LOW",
                        "pkgID": "nginx@1.27.1-1~bookworm",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2009-4487",
                        "title": "nginx: Absent sanitation of escape sequences in web server log",
                        "description": "nginx 0.7.64 writes data to a log file without sanitizing non-printable characters, which might allow remote attackers to modify a window's title, or possibly execute arbitrary commands or overwrite files, via an HTTP request containing an escape sequence for a terminal emulator.",
                        "nvdV3Score": 0,
                        "publishedDate": "2010-01-13T20:30:00.357Z"
                    },
                    {
                        "registry": "CVE-2013-0337",
                        "severity": "LOW",
                        "pkgID": "nginx@1.27.1-1~bookworm",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2013-0337",
                        "title": "The default configuration of nginx, possibly 1.3.13 and earlier, uses  ...",
                        "description": "The default configuration of nginx, possibly 1.3.13 and earlier, uses world-readable permissions for the (1) access.log and (2) error.log files, which allows local users to obtain sensitive information by reading the files.",
                        "nvdV3Score": 0,
                        "publishedDate": "2013-10-27T00:55:03.713Z"
                    },
                    {
                        "registry": "CVE-2023-44487",
                        "severity": "LOW",
                        "pkgID": "nginx@1.27.1-1~bookworm",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-44487",
                        "title": "HTTP/2: Multiple HTTP/2 enabled web servers are vulnerable to a DDoS attack (Rapid Reset Attack)",
                        "description": "The HTTP/2 protocol allows a denial of service (server resource consumption) because request cancellation can reset many streams quickly, as exploited in the wild in August through October 2023.",
                        "nvdV3Score": 7.5,
                        "publishedDate": "2023-10-10T14:15:10.883Z"
                    },
                    {
                        "registry": "CVE-2024-5535",
                        "severity": "MEDIUM",
                        "pkgID": "openssl@3.0.14-1~deb12u2",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2024-5535",
                        "title": "openssl: SSL_select_next_proto buffer overread",
                        "description": "Issue summary: Calling the OpenSSL API function SSL_select_next_proto with an\nempty supported client protocols buffer may cause a crash or memory contents to\nbe sent to the peer.\n\nImpact summary: A buffer overread can have a range of potential consequences\nsuch as unexpected application beahviour or a crash. In particular this issue\ncould result in up to 255 bytes of arbitrary private data from memory being sent\nto the peer leading to a loss of confidentiality. However, only applications\nthat directly call the SSL_select_next_proto function with a 0 length list of\nsupported client protocols are affected by this issue. This would normally never\nbe a valid scenario and is typically not under attacker control but may occur by\naccident in the case of a configuration or programming error in the calling\napplication.\n\nThe OpenSSL API function SSL_select_next_proto is typically used by TLS\napplications that support ALPN (Application Layer Protocol Negotiation) or NPN\n(Next Protocol Negotiation). NPN is older, was never standardised and\nis deprecated in favour of ALPN. We believe that ALPN is significantly more\nwidely deployed than NPN. The SSL_select_next_proto function accepts a list of\nprotocols from the server and a list of protocols from the client and returns\nthe first protocol that appears in the server list that also appears in the\nclient list. In the case of no overlap between the two lists it returns the\nfirst item in the client list. In either case it will signal whether an overlap\nbetween the two lists was found. In the case where SSL_select_next_proto is\ncalled with a zero length client list it fails to notice this condition and\nreturns the memory immediately following the client list pointer (and reports\nthat there was no overlap in the lists).\n\nThis function is typically called from a server side application callback for\nALPN or a client side application callback for NPN. In the case of ALPN the list\nof protocols supplied by the client is guaranteed by libssl to never be zero in\nlength. The list of server protocols comes from the application and should never\nnormally be expected to be of zero length. In this case if the\nSSL_select_next_proto function has been called as expected (with the list\nsupplied by the client passed in the client/client_len parameters), then the\napplication will not be vulnerable to this issue. If the application has\naccidentally been configured with a zero length server list, and has\naccidentally passed that zero length server list in the client/client_len\nparameters, and has additionally failed to correctly handle a \"no overlap\"\nresponse (which would normally result in a handshake failure in ALPN) then it\nwill be vulnerable to this problem.\n\nIn the case of NPN, the protocol permits the client to opportunistically select\na protocol when there is no overlap. OpenSSL returns the first client protocol\nin the no overlap case in support of this. The list of client protocols comes\nfrom the application and should never normally be expected to be of zero length.\nHowever if the SSL_select_next_proto function is accidentally called with a\nclient_len of 0 then an invalid memory pointer will be returned instead. If the\napplication uses this output as the opportunistic protocol then the loss of\nconfidentiality will occur.\n\nThis issue has been assessed as Low severity because applications are most\nlikely to be vulnerable if they are using NPN instead of ALPN - but NPN is not\nwidely used. It also requires an application configuration or programming error.\nFinally, this issue would not typically be under attacker control making active\nexploitation unlikely.\n\nThe FIPS modules in 3.3, 3.2, 3.1 and 3.0 are not affected by this issue.\n\nDue to the low severity of this issue we are not issuing new releases of\nOpenSSL at this time. The fix will be included in the next releases when they\nbecome available.",
                        "nvdV3Score": 0,
                        "publishedDate": "2024-06-27T11:15:24.447Z"
                    },
                    {
                        "registry": "CVE-2023-4641",
                        "severity": "MEDIUM",
                        "pkgID": "passwd@1:4.13+dfsg1-1+b1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-4641",
                        "title": "shadow-utils: possible password leak during passwd(1) change",
                        "description": "A flaw was found in shadow-utils. When asking for a new password, shadow-utils asks the password twice. If the password fails on the second attempt, shadow-utils fails in cleaning the buffer used to store the first entry. This may allow an attacker with enough access to retrieve the password from the memory.",
                        "nvdV3Score": 5.5,
                        "publishedDate": "2023-12-27T16:15:13.363Z"
                    },
                    {
                        "registry": "CVE-2007-5686",
                        "severity": "LOW",
                        "pkgID": "passwd@1:4.13+dfsg1-1+b1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2007-5686",
                        "title": "initscripts in rPath Linux 1 sets insecure permissions for the /var/lo ...",
                        "description": "initscripts in rPath Linux 1 sets insecure permissions for the /var/log/btmp file, which allows local users to obtain sensitive information regarding authentication attempts.  NOTE: because sshd detects the insecure permissions and does not log certain events, this also prevents sshd from logging failed authentication attempts by remote attackers.",
                        "nvdV3Score": 0,
                        "publishedDate": "2007-10-28T17:08:00Z"
                    },
                    {
                        "registry": "CVE-2019-19882",
                        "severity": "LOW",
                        "pkgID": "passwd@1:4.13+dfsg1-1+b1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2019-19882",
                        "title": "shadow-utils: local users can obtain root access because setuid programs are misconfigured",
                        "description": "shadow 4.8, in certain circumstances affecting at least Gentoo, Arch Linux, and Void Linux, allows local users to obtain root access because setuid programs are misconfigured. Specifically, this affects shadow 4.8 when compiled using --with-libpam but without explicitly passing --disable-account-tools-setuid, and without a PAM configuration suitable for use with setuid account management tools. This combination leads to account management tools (groupadd, groupdel, groupmod, useradd, userdel, usermod) that can easily be used by unprivileged local users to escalate privileges to root in multiple ways. This issue became much more relevant in approximately December 2019 when an unrelated bug was fixed (i.e., the chmod calls to suidusbins were fixed in the upstream Makefile which is now included in the release version 4.8).",
                        "nvdV3Score": 7.8,
                        "publishedDate": "2019-12-18T16:15:26.963Z"
                    },
                    {
                        "registry": "CVE-2023-29383",
                        "severity": "LOW",
                        "pkgID": "passwd@1:4.13+dfsg1-1+b1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-29383",
                        "title": "shadow: Improper input validation in shadow-utils package utility chfn",
                        "description": "In Shadow 4.13, it is possible to inject control characters into fields provided to the SUID program chfn (change finger). Although it is not possible to exploit this directly (e.g., adding a new user fails because \\n is in the block list), it is possible to misrepresent the /etc/passwd file when viewed. Use of \\r manipulations and Unicode characters to work around blocking of the : character make it possible to give the impression that a new user has been added. In other words, an adversary may be able to convince a system administrator to take the system offline (an indirect, social-engineered denial of service) by demonstrating that \"cat /etc/passwd\" shows a rogue user account.",
                        "nvdV3Score": 3.3,
                        "publishedDate": "2023-04-14T22:15:07.68Z"
                    },
                    {
                        "registry": "TEMP-0628843-DBAD28",
                        "severity": "LOW",
                        "pkgID": "passwd@1:4.13+dfsg1-1+b1",
                        "primaryURL": "https://security-tracker.debian.org/tracker/TEMP-0628843-DBAD28",
                        "title": "[more related to CVE-2005-4890]",
                        "description": "",
                        "nvdV3Score": 0,
                        "publishedDate": null
                    },
                    {
                        "registry": "CVE-2023-31484",
                        "severity": "HIGH",
                        "pkgID": "perl-base@5.36.0-7+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-31484",
                        "title": "perl: CPAN.pm does not verify TLS certificates when downloading distributions over HTTPS",
                        "description": "CPAN.pm before 2.35 does not verify TLS certificates when downloading distributions over HTTPS.",
                        "nvdV3Score": 8.1,
                        "publishedDate": "2023-04-29T00:15:09Z"
                    },
                    {
                        "registry": "CVE-2011-4116",
                        "severity": "LOW",
                        "pkgID": "perl-base@5.36.0-7+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2011-4116",
                        "title": "perl: File:: Temp insecure temporary file handling",
                        "description": "_is_safe in the File::Temp module for Perl does not properly handle symlinks.",
                        "nvdV3Score": 7.5,
                        "publishedDate": "2020-01-31T18:15:11.343Z"
                    },
                    {
                        "registry": "CVE-2023-31486",
                        "severity": "LOW",
                        "pkgID": "perl-base@5.36.0-7+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-31486",
                        "title": "http-tiny: insecure TLS cert default",
                        "description": "HTTP::Tiny before 0.083, a Perl core module since 5.13.9 and available standalone on CPAN, has an insecure default TLS configuration where users must opt in to verify certificates.",
                        "nvdV3Score": 8.1,
                        "publishedDate": "2023-04-29T00:15:09.083Z"
                    },
                    {
                        "registry": "TEMP-0517018-A83CE6",
                        "severity": "LOW",
                        "pkgID": "sysvinit-utils@3.06-4",
                        "primaryURL": "https://security-tracker.debian.org/tracker/TEMP-0517018-A83CE6",
                        "title": "[sysvinit: no-root option in expert installer exposes locally exploitable security flaw]",
                        "description": "",
                        "nvdV3Score": 0,
                        "publishedDate": null
                    },
                    {
                        "registry": "CVE-2005-2541",
                        "severity": "LOW",
                        "pkgID": "tar@1.34+dfsg-1.2+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2005-2541",
                        "title": "tar: does not properly warn the user when extracting setuid or setgid files",
                        "description": "Tar 1.15.1 does not properly warn the user when extracting setuid or setgid files, which may allow local users or remote attackers to gain privileges.",
                        "nvdV3Score": 0,
                        "publishedDate": "2005-08-10T04:00:00Z"
                    },
                    {
                        "registry": "TEMP-0290435-0B57B5",
                        "severity": "LOW",
                        "pkgID": "tar@1.34+dfsg-1.2+deb12u1",
                        "primaryURL": "https://security-tracker.debian.org/tracker/TEMP-0290435-0B57B5",
                        "title": "[tar's rmt command may have undesired side effects]",
                        "description": "",
                        "nvdV3Score": 0,
                        "publishedDate": null
                    },
                    {
                        "registry": "CVE-2022-0563",
                        "severity": "LOW",
                        "pkgID": "util-linux@2.38.1-5+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2022-0563",
                        "title": "util-linux: partial disclosure of arbitrary files in chfn and chsh when compiled with libreadline",
                        "description": "A flaw was found in the util-linux chfn and chsh utilities when compiled with Readline support. The Readline library uses an \"INPUTRC\" environment variable to get a path to the library config file. When the library cannot parse the specified file, it prints an error message containing data from the file. This flaw allows an unprivileged user to read root-owned files, potentially leading to privilege escalation. This flaw affects util-linux versions prior to 2.37.4.",
                        "nvdV3Score": 5.5,
                        "publishedDate": "2022-02-21T19:15:08.393Z"
                    },
                    {
                        "registry": "CVE-2022-0563",
                        "severity": "LOW",
                        "pkgID": "util-linux-extra@2.38.1-5+deb12u1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2022-0563",
                        "title": "util-linux: partial disclosure of arbitrary files in chfn and chsh when compiled with libreadline",
                        "description": "A flaw was found in the util-linux chfn and chsh utilities when compiled with Readline support. The Readline library uses an \"INPUTRC\" environment variable to get a path to the library config file. When the library cannot parse the specified file, it prints an error message containing data from the file. This flaw allows an unprivileged user to read root-owned files, potentially leading to privilege escalation. This flaw affects util-linux versions prior to 2.37.4.",
                        "nvdV3Score": 5.5,
                        "publishedDate": "2022-02-21T19:15:08.393Z"
                    },
                    {
                        "registry": "CVE-2023-45853",
                        "severity": "CRITICAL",
                        "pkgID": "zlib1g@1:1.2.13.dfsg-1",
                        "primaryURL": "https://avd.aquasec.com/nvd/cve-2023-45853",
                        "title": "zlib: integer overflow and resultant heap-based buffer overflow in zipOpenNewFileInZip4_6",
                        "description": "MiniZip in zlib through 1.3 has an integer overflow and resultant heap-based buffer overflow in zipOpenNewFileInZip4_64 via a long filename, comment, or extra field. NOTE: MiniZip is not a supported part of the zlib product. NOTE: pyminizip through 0.2.6 is also vulnerable because it bundles an affected zlib version, and exposes the applicable MiniZip code through its compress API.",
                        "nvdV3Score": 9.8,
                        "publishedDate": "2023-10-14T02:15:09.323Z"
                    }
                ]
            },
            "secrets": {
                "unknown": 0,
                "low": 0,
                "medium": 0,
                "high": 0,
                "critical": 0
            }
        }
    }
}
```
