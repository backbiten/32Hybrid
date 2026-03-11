import argparse
import subprocess
import os
import sys

def decrypt_file(infile, outfile, homedir=None):
    command = ["gpg", "--output", outfile, "--decrypt", infile]
    
    if homedir:
        command.insert(0, f'--homedir={homedir}') 
    
    subprocess.run(command, check=True)

def main():
    parser = argparse.ArgumentParser(description='Import OpenPGP file.')
    parser.add_argument('infile', help='Input GPG file to decrypt')
    parser.add_argument('outfile', help='Output plaintext file')
    parser.add_argument('--homedir', help='Specify an alternate GnuPG home directory')
    
    args = parser.parse_args()
    
    decrypt_file(args.infile, args.outfile, args.homedir)

if __name__ == '__main__':
    main()
