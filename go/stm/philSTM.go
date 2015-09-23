package main

import (
  "./stm"
  "time"
  "fmt"
)

type Stick struct {
  tvar *stm.TVar
}

func NewStick() *Stick {
  return &Stick{
    tvar: stm.NewTVar(false),
  }
}

func (s *Stick) take(state *stm.RWSet) error {
  value, err := s.tvar.Get(state)
  taken := value.(bool)

  if err != nil {
    return err
  }

  if (taken) {
    return stm.Retry()
  }

  s.tvar.Set(true, state)

  return nil
}

func (s *Stick) put(state *stm.RWSet) error {
  s.tvar.Set(false, state)
  return nil
}

func phil(s1,s2 *Stick, name string) {
  for {
    take := stm.Atomically()
    take.SetAction(func() (stm.STMValue, error) {
      err := s1.take(take.GetState())

      if err != nil {
        // fmt.Printf("%s in %s.\n", err.Error(), name)
        return nil, err
      }

      err = s2.take(take.GetState())

      if err != nil {
        // fmt.Printf("%s in %s.\n", err.Error(), name)
        return nil, err
      }

      return nil, nil
    })
    take.Execute()

    fmt.Println(name + " is eating...")
    time.Sleep(1 * time.Second)

    put := stm.Atomically()
    put.SetAction(func() (stm.STMValue, error) {
      s1.put(put.GetState())
      s2.put(put.GetState())
      return nil, nil
    })
    put.Execute()

    fmt.Println(name + " put back...")
    time.Sleep(1 * time.Second)
  }
}

func main() {
  s1 := NewStick()
  s2 := NewStick()
  s3 := NewStick()

  go phil(s1, s2, "1")
  go phil(s2, s3, "2")
  phil (s3, s1, "3")
}
