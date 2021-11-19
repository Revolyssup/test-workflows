# Copyright 2019 Layer5 Authors
#
#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at
#
#       http://www.apache.org/licenses/LICENSE-2.0
#
#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.

#!/usr/bin/env bash

# Get the platform to deploy Meshery from the flag
# Get the deploy option on whether to install cli alone or to deploy Meshery server as well
# Default option is to start Meshery server

while getopts ":np:" opt; do
    case $opt in
    n)
        DEPLOY_MESHERY=false
        ;;
    p)
        case $OPTARG in
        docker)
            PLATFORM="docker"
            ;;
        kubernetes)
            PLATFORM="kubernetes"
            ;;
        *)
            echo "Invalid platform: $OPTARG"
            exit 1
            ;;
        esac
        ;;
    \?)
        echo "Invalid option: -$OPTARG" >&2
        exit 1
        ;;
    :)
        echo "Option -$OPTARG requires a platform argument." >&2
        exit 1
        ;;
    esac
done

# If user wants to install the cli and deploy meshery as well, read the platform flag
if [ "$DEPLOY_MESHERY" != false ]; then
  # If no platform flag is present, prompt the user for a platform
  if [ ! "$PLATFORM" ]; then
      echo Enter a platform to deploy Meshery. Available platforms [docker, kubernetes]:
      read PLATFORM < /dev/tty
  fi

  case "$PLATFORM" in
  docker | kubernetes) ;;

  *)
      echo "Invalid platform: $PLATFORM"
      exit 1
      ;;
  esac
fi


####### COMMON FUNCTIONS
############################
command_exists() {
    command -v $1 >/dev/null 2>&1
}

#######   IDENTIFY OS
############################

OSARCHITECTURE="$(uname -m)"
OS="$(uname)"

if [ "x${OS}" = "xDarwin" ] ; then
  OSEXT="Darwin"
else
  OSEXT="Linux"
fi

#######   PREFLIGHT CHECK
############################

if ! command_exists curl ; then
    echo "Missing required utility: 'curl'. Please install 'curl' and try again."
    exit;
fi

if ! command_exists unzip ; then
    echo "Missing required utility: 'unzip'. Please install 'unzip' and try again."
    exit;
fi

if [ "x${MESHERY_VERSION}" = "x" ] ; then
  MESHERY_VERSION=$(curl -L -s https://github.com/meshery/meshery/releases/latest | \
                  grep "/releases/tag/" | head -1 | awk -F '"' '{print $4}' | \
                  awk -F '/' '{print $NF}' )
fi

if [ "x${MESHERY_VERSION}" = "x" ] ; then
  printf "Unable to get latest mesheryctl version. Set MESHERY_VERSION env var and re-run. For example: export MESHERY_VERSION=v0.1.6\n"
  exit;
fi

NAME="mesheryctl-${MESHERY_VERSION}"
URL="https://github.com/meshery/meshery/releases/download/${MESHERY_VERSION}/mesheryctl_${MESHERY_VERSION:1}_${OSEXT}_${OSARCHITECTURE}.zip"

printf "\nDownloading %s for %s...\n\n" "$NAME" "$OSEXT"
curl -L ${URL} -o ${PWD}/meshery.zip

validFile=`file ${PWD}/meshery.zip | grep 'Zip archive data'`
if [ -z "$validFile" ] ; then
  printf "Unable to get latest meshery install package. Set MESHERY_VERSION env var and re-run. For example: export MESHERY_VERSION=v0.1.6\n"
  exit;
fi

# Generate a temporary folder to store intermediate installation files
temporary_dir_name=$(tr -dc A-Za-z0-9 </dev/urandom 2>/dev/null | head -c 16)
if [ -z "$temporary_dir_name" ] ; then
   temporary_dir_name="temp_extract_dir"
fi

mkdir $temporary_dir_name
if [ $? -ne 0 ] ; then
  rm ${PWD}/meshery.zip
  printf "Disk space is low on the system, Re-try installation after cleaning up some files.\n"
  exit 1;
fi

printf "\nExtracting %s to temporary folder %s...\n" "$NAME" "$temporary_dir_name"
unzip ${PWD}/meshery.zip -d ${PWD}/$temporary_dir_name

# Use user provided installation path from env variable(MESHERY_INSTALL_DIR)
# by default its /usr/local/bin
install_dir=${MESHERY_INSTALL_DIR}
if [ -z "$install_dir" ]
then
    install_dir="/usr/local/bin"
fi

printf "\nInstalling mesheryctl in $install_dir.\n"
WHOAMI=$(whoami)
if mv ${PWD}/$temporary_dir_name/mesheryctl $install_dir/mesheryctl >/dev/null 2>&1; then
  echo "mesheryctl installed."
else
  if sudo mv ${PWD}/$temporary_dir_name/mesheryctl $install_dir/mesheryctl ; then
    echo "Permission Resolved: mesheryctl installed using sudo permissions."
  else
    echo "Cannot install mesheryctl. Check permissions of $WHOAMI for $install_dir."
    exit 1
  fi
fi

#Transfering permissions and ownership to client USER
if [ "$SUDO_USER" == "" ] ;
then
  if chown $WHOAMI $install_dir/mesheryctl ; then
    # Give write access to user to create .meshery folder in home directory
    chown $WHOAMI $HOME
    chmod u+w $HOME
    echo "permissions moved to "$WHOAMI
  else
    echo "Unable to write to $install_dir. Please verify write permission for $WHOAMI."
    exit 1
  fi
else
  if chown $SUDO_USER /usr/local/bin/mesheryctl ; then
    # Give write access to SUDO_USER to create .meshery folder in home directory
    chown $SUDO_USER $HOME
    chmod u+w $HOME
    echo "permissions moved to "$SUDO_USER
  else
    echo "Unable to write to $install_dir/mesheryctl. Please verify write permission for $SUDO_USER."
    exit 1
  fi
fi


printf "Removing installation files"
rm -rf meshery.zip ${PWD}/$temporary_dir_name/

if [ "$DEPLOY_MESHERY" == false ] ;
then
    printf "\nRun \"mesheryctl system start\" to start Meshery.\n"
    exit 0
fi

# if ADAPTERS are empty then we'd use default local context else go we'd create custom context, which only fetches required adapters
# Valid adapter option are
# cpx
# consul
# istio
# linkerd
# nsm
# kuma
# cpx
# osm
# traefik
# nginx
# tanzu
# app

## Any adapter other than above would be ignored
if [ ! -z "$ADAPTERS" ]; then
    adpaterString=
    for adapter in $ADAPTERS; do
        case $adapter in
            cpx|consul|istio|linkerd|nsm|kuma|cpx|osm)
                adapterString="$adapterString -a meshery-$adapter "
            ;;
            nginx|tanzu)
                adapterString="$adapterString -a meshery-$adapter-sm "
            ;;
            traefik|app)
                adapterString="$adapterString -a meshery-$adapter-mesh "
            ;;
            *)
                # We are just neglecting invalid mesh options
            ;;
        esac    
    done

    mesheryctl system context create custom -s $adapterString --yes
fi

printf "\nStarting Meshery...\n"
mesheryctl system start --yes --platform $PLATFORM
