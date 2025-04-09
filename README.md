
# BPM Utils
### _Creation and maintenance utilities for BPM packages and repositories_ ###

## Information
BPM Utils provides a number of different helper commands for creating and maintaining BPM packages or repositories

## Provided utilities
- bpm-setup (Sets up directories for BPM source package creation)
- bpm-repo (Allows for easy management of multiple-package repositories)
- bpm-package (Turns a BPM package directory into a .bpm archive)
- bpm-convert (Converts source packages to binary ones)

## Installation
#### Using a package manager
- Tide Linux: Tide linux provides a `bpm-utils` package which can be installed using `bpm install bpm-utils`
#### Building from source
- Download `go` from your package manager or from the go website
- Download `make` from your package manager
- Run the following command to compile the project
```
make SYSCONFDIR=/etc
```
- Run the following command to install bpm-utils to your system. You may also append a DESTDIR variable at the end of this line if you wish to install the files to a different location
```
make install PREFIX=/usr SYSCONFDIR=/etc
```
## Package Creation using BPM Utils
Creating a package for BPM with these utilities is simple

1) Run the following command (You can run the command with no arguments to see all available options)
```
bpm-setup -D my_package
```
2) This will create a directory named `my_package` containing all files required for bpm package creation
3) You may wish to edit the pkg.info descriptor file inside the newly created directory to include dependencies or add/change other information. Here's an example of what a descriptor  file could look like
```yaml
name: my_package
description: My package's description
version: 1.0
revision: 2 (Optional)
url: https://www.my-website.com/ (Optional)
license: MyLicense (Optional)
architecture: x86_64
depends: ["dependency1","dependency2"] (Optional)
optional_depends: ["optional_depend1","optional_depend2"] (Optional)
make_depends: ["make_depend1","make_depend2"] (Optional)
keep: ["etc/my_config.conf"] (Optional)
type: source
```

4) If you would like to bundle patches or other files with your package place them in the 'source-files' directory. They will be extracted to the same location as the source.sh file during compilation
5) You now need to edit your source.sh file which contains the compilation instructions for your package, the default source template comments should explain the basic process of compiling your program and how to edit it
6) When you are done editing your source.sh script run the following command to create a BPM package archive. You may run the `bpm-package` command with no arguments to get an explanation of what each flag does
```
bpm-package -d
```
7) The `bpm-package` command will output a binary bpm archive which can be installed by BPM using `bpm install <file.bpm>`. If you are operating inside a BPM repository created using `bpm-repo` the file will automatically be moved to the binary subdirectory of your package repository
