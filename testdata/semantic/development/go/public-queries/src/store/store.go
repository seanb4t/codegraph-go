package store

func Load() int { return read() }

func read() int { return 1 }

type Disk struct{}

func (Disk) Same() int { return 1 }

type Memory struct{}

func (Memory) Same() int { return 2 }
