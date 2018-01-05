/*
	This package is to provide the basic mechanisms,
	which are the fundations for building the high reliable microservice application

	Author: Chao Cai

*/
package microservice_helper

import (
	"strings"
	"time"

	"github.com/afex/hystrix-go/hystrix"
)

type FallbackFunc func(error) (interface{}, error)

type RetrySettings struct {
	retryTimes             int           //retry times befpre returning error
	retryInterval          time.Duration //the initial interval for retrying
	retryIntervalIncrement time.Duration //the interval would be increased by the value
}

func isRetryable(err error, retryableErrorFlags []string) bool {
	for _, flag := range retryableErrorFlags {
		if strings.Contains(err.Error(), flag) {
			return true
		}
	}
	return false
}

//call dependent service with the fallback and circuit mechanism
func CallDependentService(settingGroup string, //configuration setting group name
	invokeDependentService func() (interface{}, error),
	fallbackFunc FallbackFunc) (interface{}, error) {
	output := make(chan interface{}, 1)
	ret := interface{}(nil)
	err := error(nil)
	errors := hystrix.Go(settingGroup, func() error {
		defer close(output)
		ret, err = invokeDependentService()

		if err == nil {
			output <- ret
			return nil
		}
		return err
	}, nil)

	select {
	case ret = <-output:
		if err == nil || fallbackFunc == nil {
			return ret, err
		}
		return fallbackFunc(err)
	case err = <-errors:
		if fallbackFunc == nil {
			return ret, err
		}
		return fallbackFunc(err)
	}
}

func isRetryableError(err error, retryableErrors *[]error) bool {
	for _, retryableError := range *retryableErrors {
		if err == retryableError {
			return true
		}
	}
	return false
}

//retry the logic when retryable errors are thrown
func AutoRetry(runnable func() (interface{}, error), retrySettings RetrySettings,
	retryableErrors []error) (interface{}, error) {
	var ret interface{}
	var err error
	retryInterval := retrySettings.retryInterval
	i := 0
	for {
		if ret, err = runnable(); err == nil {
			return ret, nil
		}

		if !(isRetryableError(err, &retryableErrors)) {
			break
		}

		if i >= retrySettings.retryTimes {
			break
		}
		time.Sleep(retryInterval)
		retryInterval += retrySettings.retryIntervalIncrement
		i = i + 1
	}
	return ret, err
}
