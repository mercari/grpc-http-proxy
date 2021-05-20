/*
* created by Roman Zhyliov <roman@kintohub.com>
*/

package utils

import (
    "os"
)

func GetEnvVar(key, fallback string) string {

    returnVal := fallback

    if value, ok := os.LookupEnv(key); ok {
        returnVal = value
    }
    return returnVal
}