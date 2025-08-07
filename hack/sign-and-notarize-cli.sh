#!/bin/bash

echo "checking required environment variables are set..."

if [[ -z "${KARGO_BIN_PATH}" ]]; then
    echo "KARGO_BIN_PATH is unset." && exit 1
fi

if [[ -z "${QUILL_SIGN_P12}" ]]; then
    echo "QUILL_SIGN_P12 is unset." && exit 1
fi

if [[ -z "${QUILL_SIGN_PASSWORD}" ]]; then
    echo "QUILL_SIGN_PASSWORD is unset." && exit 1
fi

if [[ -z "${QUILL_NOTARY_KEY}" ]]; then
    echo "QUILL_NOTARY_KEY is unset." && exit 1
fi

if [[ -z "${QUILL_NOTARY_KEY_ID}" ]]; then
    echo "QUILL_NOTARY_KEY_ID is unset." && exit 1
fi

if [[ -z "${QUILL_NOTARY_ISSUER}" ]]; then
    echo "QUILL_NOTARY_ISSUER is unset." && exit 1
fi

echo "confirmed environment variables are set correctly."

if [[ ! -x "$(command -v quill)" ]]; then
    echo "quill is not installed. Installing..."
    curl -sSfL https://get.anchore.io/quill | sh -s -- -b /usr/local/bin \
    && echo "quill installed successfully."
else
    echo "quill is already installed."
fi

echo "signing and notarizing $KARGO_BIN_PATH"

quill sign-and-notarize --p12 $QUILL_SIGN_P12 $KARGO_BIN_PATH
if [ $? -ne 0 ]; then
    echo "failed to sign and notarize $KARGO_BIN_PATH" && exit 1
fi

echo "successfully signed and notarized $KARGO_BIN_PATH"