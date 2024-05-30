
# BPM Utils
### _Package creation and editing utilities for BPM_ ###

## Information

BPM Utils is a package providing a number of different helper scripts for setting up and archiving BPM packages

## Provided Scripts
- bpm-setup (Creates a directory with the required files for a BPM package)
-  bpm-package (Turns a BPM package directory into a .bpm archive)

## Installation

Currently all BPM Utilities are simple bash scripts. This means you are able to simply clone this repository and place these scripts wherever you would like. Additionally pre-made packages are available for the following package managers: \
BPM: https://gitlab.com/bubble-package-manager/bpm-utils-bpm \
Pacman: https://gitlab.com/bubble-package-manager/bpm-utils-pacman

## Package Creation using BPM Utils

Creating a package for BPM with these utilities is simple

2) Run the following command (You can run the comamnd with no arguments to see available options)
```
bpm-setup -D my_bpm_package -t <binary/source>
```
3) This will create a directory named 'my_bpm_package' under the current directory with all the required files for the chosen package type
4) You are able to edit the pkg.info descriptor file inside the newly created directory to your liking. Here's an example of what a descriptor  file could look like
```
name: my_package
description: My package's description
version: 1.0
architecture: x86_64
url: https://www.my-website.com/ (Optional)
license: MyLicense (Optional)
type: <binary/source>
depends: dependency1,dependency2 (Optional)
make_depends: make_depend1,make_depend2 (Optional)
```
### Binary Packages
3) If you are making a binary package, copy all your binaries along with the directories they reside in (i.e files/usr/bin/my_binary)
6) Run the following to create a package archive
```
bpm-package <filename.bpm>
```
7) It's done! You now hopefully have a working BPM package!
### Source Packages
3) If you would like to bundle patches or other files with your source package place them in the 'source-files' directory. They will be extracted to the same location as the source.sh file during compilation
4) You need to edit your 'source.sh' file, the default source.sh template should explain the basic process of compiling your program
5) Your goal is to download your program's source code with either git, wget, curl, etc. and put the binaries under a folder called 'output' in the root of the temp directory. There is a simple example script with helpful comments in the htop-src test package
6) When you are done making your source.sh script run the following to create a package archive. You may also append the -c flag to compile the package and create a binary package as well
```
bpm-package <filename.bpm>
```
7) That's it! Your source package should now be compiling correctly!
