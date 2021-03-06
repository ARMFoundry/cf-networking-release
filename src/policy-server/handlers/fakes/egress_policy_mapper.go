// Code generated by counterfeiter. DO NOT EDIT.
package fakes

import (
	"policy-server/store"
	"sync"
)

type EgressPolicyMapper struct {
	AsStoreEgressPolicyStub        func(bytes []byte) ([]store.EgressPolicy, error)
	asStoreEgressPolicyMutex       sync.RWMutex
	asStoreEgressPolicyArgsForCall []struct {
		bytes []byte
	}
	asStoreEgressPolicyReturns struct {
		result1 []store.EgressPolicy
		result2 error
	}
	asStoreEgressPolicyReturnsOnCall map[int]struct {
		result1 []store.EgressPolicy
		result2 error
	}
	AsBytesStub        func(storeEgressPolicies []store.EgressPolicy) ([]byte, error)
	asBytesMutex       sync.RWMutex
	asBytesArgsForCall []struct {
		storeEgressPolicies []store.EgressPolicy
	}
	asBytesReturns struct {
		result1 []byte
		result2 error
	}
	asBytesReturnsOnCall map[int]struct {
		result1 []byte
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *EgressPolicyMapper) AsStoreEgressPolicy(bytes []byte) ([]store.EgressPolicy, error) {
	var bytesCopy []byte
	if bytes != nil {
		bytesCopy = make([]byte, len(bytes))
		copy(bytesCopy, bytes)
	}
	fake.asStoreEgressPolicyMutex.Lock()
	ret, specificReturn := fake.asStoreEgressPolicyReturnsOnCall[len(fake.asStoreEgressPolicyArgsForCall)]
	fake.asStoreEgressPolicyArgsForCall = append(fake.asStoreEgressPolicyArgsForCall, struct {
		bytes []byte
	}{bytesCopy})
	fake.recordInvocation("AsStoreEgressPolicy", []interface{}{bytesCopy})
	fake.asStoreEgressPolicyMutex.Unlock()
	if fake.AsStoreEgressPolicyStub != nil {
		return fake.AsStoreEgressPolicyStub(bytes)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.asStoreEgressPolicyReturns.result1, fake.asStoreEgressPolicyReturns.result2
}

func (fake *EgressPolicyMapper) AsStoreEgressPolicyCallCount() int {
	fake.asStoreEgressPolicyMutex.RLock()
	defer fake.asStoreEgressPolicyMutex.RUnlock()
	return len(fake.asStoreEgressPolicyArgsForCall)
}

func (fake *EgressPolicyMapper) AsStoreEgressPolicyArgsForCall(i int) []byte {
	fake.asStoreEgressPolicyMutex.RLock()
	defer fake.asStoreEgressPolicyMutex.RUnlock()
	return fake.asStoreEgressPolicyArgsForCall[i].bytes
}

func (fake *EgressPolicyMapper) AsStoreEgressPolicyReturns(result1 []store.EgressPolicy, result2 error) {
	fake.AsStoreEgressPolicyStub = nil
	fake.asStoreEgressPolicyReturns = struct {
		result1 []store.EgressPolicy
		result2 error
	}{result1, result2}
}

func (fake *EgressPolicyMapper) AsStoreEgressPolicyReturnsOnCall(i int, result1 []store.EgressPolicy, result2 error) {
	fake.AsStoreEgressPolicyStub = nil
	if fake.asStoreEgressPolicyReturnsOnCall == nil {
		fake.asStoreEgressPolicyReturnsOnCall = make(map[int]struct {
			result1 []store.EgressPolicy
			result2 error
		})
	}
	fake.asStoreEgressPolicyReturnsOnCall[i] = struct {
		result1 []store.EgressPolicy
		result2 error
	}{result1, result2}
}

func (fake *EgressPolicyMapper) AsBytes(storeEgressPolicies []store.EgressPolicy) ([]byte, error) {
	var storeEgressPoliciesCopy []store.EgressPolicy
	if storeEgressPolicies != nil {
		storeEgressPoliciesCopy = make([]store.EgressPolicy, len(storeEgressPolicies))
		copy(storeEgressPoliciesCopy, storeEgressPolicies)
	}
	fake.asBytesMutex.Lock()
	ret, specificReturn := fake.asBytesReturnsOnCall[len(fake.asBytesArgsForCall)]
	fake.asBytesArgsForCall = append(fake.asBytesArgsForCall, struct {
		storeEgressPolicies []store.EgressPolicy
	}{storeEgressPoliciesCopy})
	fake.recordInvocation("AsBytes", []interface{}{storeEgressPoliciesCopy})
	fake.asBytesMutex.Unlock()
	if fake.AsBytesStub != nil {
		return fake.AsBytesStub(storeEgressPolicies)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.asBytesReturns.result1, fake.asBytesReturns.result2
}

func (fake *EgressPolicyMapper) AsBytesCallCount() int {
	fake.asBytesMutex.RLock()
	defer fake.asBytesMutex.RUnlock()
	return len(fake.asBytesArgsForCall)
}

func (fake *EgressPolicyMapper) AsBytesArgsForCall(i int) []store.EgressPolicy {
	fake.asBytesMutex.RLock()
	defer fake.asBytesMutex.RUnlock()
	return fake.asBytesArgsForCall[i].storeEgressPolicies
}

func (fake *EgressPolicyMapper) AsBytesReturns(result1 []byte, result2 error) {
	fake.AsBytesStub = nil
	fake.asBytesReturns = struct {
		result1 []byte
		result2 error
	}{result1, result2}
}

func (fake *EgressPolicyMapper) AsBytesReturnsOnCall(i int, result1 []byte, result2 error) {
	fake.AsBytesStub = nil
	if fake.asBytesReturnsOnCall == nil {
		fake.asBytesReturnsOnCall = make(map[int]struct {
			result1 []byte
			result2 error
		})
	}
	fake.asBytesReturnsOnCall[i] = struct {
		result1 []byte
		result2 error
	}{result1, result2}
}

func (fake *EgressPolicyMapper) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.asStoreEgressPolicyMutex.RLock()
	defer fake.asStoreEgressPolicyMutex.RUnlock()
	fake.asBytesMutex.RLock()
	defer fake.asBytesMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *EgressPolicyMapper) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}
