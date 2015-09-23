package stm

import (
  "errors"
  "log"
  // "io/ioutil"
)

type STMValue interface {
}

type TVar struct {
  holder chan STMValue
}

func NewTVar(value STMValue) *TVar {
  retval := &TVar{
    holder: make(chan STMValue, 1),
  }

  retval.holder <- value

  return retval
}

func (t *TVar) Get() (STMValue, error) {
  if state.ws[t] == nil {
    value := <- t.holder
    t.holder <- value

    if state.rs[t] == nil {
      state.rs[t] = value
      return value, nil
    } else {
      if value != state.rs[t] {
        return nil, errors.New("rollback")
      } else {
        return value, nil
      }
    }
  } else {
    return state.ws[t], nil
  }
}

func (t *TVar) Set(v STMValue) {
  state.ws[t] = v
}

type RWSet struct {
  rs map[*TVar]STMValue
  ws map[*TVar]STMValue
}

func NewRWSet() *RWSet {
  return &RWSet{
    rs: make(map[*TVar]STMValue),
    ws: make(map[*TVar]STMValue),
  }
}

type Action func() (STMValue, error)

func lockState(state *RWSet) map[*TVar]STMValue {
  // lock all tvars and remember values in them.
  storage := make(map[*TVar]STMValue)

  for t, _ := range state.ws {
    _, exists := storage[t]
    // do not try to double-lock a tvar :-(
    if !exists {
      storage[t] = <- t.holder
    }
  }

  for t, _ := range state.rs {
    _, exists := storage[t]
    // do not try to double-lock a tvar :-(
    if !exists {
      storage[t] = <- t.holder
    }
  }

  return storage
}

func validate (storage map[*TVar]STMValue, state *RWSet) bool {
  log.Println(storage)
  log.Println(state)
  // validate readset.
  for t, v := range state.rs {
    if storage[t] != v {
      return false;
    }
  }
  return true;
}

func Retry() error {
  return errors.New("retry")
}

var state *RWSet = NewRWSet()
var notifier chan bool = make(chan bool, 1)
var retryState bool = false

type AtomicallyType struct {
  trans Action
  state *RWSet
  notifier chan bool
  retryState bool
}

func Atomically2(action Action) *AtomicallyType {
  return &AtomicallyType{
    trans: action,
    state: NewRWSet(),
    notifier: make(chan bool, 1),
    retryState: false,
  }
}

func (a *AtomicallyType) execute() {

}


func Atomically (trans Action) STMValue {
  // log.SetOutput(ioutil.Discard)

  state = NewRWSet() // clear rw-set.

  log.Println("===================")
  log.Println("Atomically start..")
  log.Println(state)

  log.Println("Executing Trans..")
  result, err := trans()

  log.Println(result)
  log.Println(err)

  if err != nil {
    switch err.Error() {
    case "rollback":
      log.Println("Executing rollback!")
      log.Println("===================")
      return Atomically(trans)
    case "retry":
      retryState = true
      <- notifier
      retryState = false
      log.Println("Apply Retry...")
      log.Println("===================")
      return Atomically(trans)
    }
  }

  log.Println("Locking State.")
  storage := lockState(state)

  log.Println("Validating ...")
  valid := validate(storage, state)
  log.Println(valid)

  if !valid {
    // unlock and reset to old values
    for k, v := range storage {
      k.holder <- v
    }
    log.Println("Not Valid! Resetting...")
    log.Println("===================")
    return Atomically(trans)
  } else {
    // commit
    for key, value := range(state.ws) {
      storage[key] = value
    }

    // unlock
    for k, v := range storage {
      k.holder <- v
    }

    if retryState {
      notifier <- true
    }

    log.Println("Successful Transaction!")
    log.Println("===================")
    return result
  }
}