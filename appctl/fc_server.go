package appctl

import (
	"context"
	"fmt"
	"github.com/t-beigbeder/otvl_qstf/stf"
	"log/slog"
)

func (ac *appServerCnc) newFunction(id string, fc stf.Function, fd *FunctionDesc, sths *StreamHandlers) error {
	ac.mux.Lock()
	defer ac.mux.Unlock()
	_, ok := ac.funcs[id]
	if ok {
		return fmt.Errorf("function id %s already exists", id)
	}
	ac.cnc.GetLogger().Info("new function", "id", id)
	ac.funcs[id] = fc
	ac.fds[id] = fd
	ac.sthss[id] = sths
	return nil
}

func (ac *appServerCnc) delFunction(id string) error {
	ac.mux.Lock()
	defer ac.mux.Unlock()
	fc, ok := ac.funcs[id]
	if !ok {
		return fmt.Errorf("function id %s does not exist", id)
	}
	ac.cnc.GetLogger().Info("delete function", "id", id)
	defer func() {
		delete(ac.funcs, id)
		delete(ac.fds, id)
		delete(ac.sthss, id)
	}()
	fc.Close()
	return nil

}

func (ac *appServerCnc) getFunction(id string) (stf.Function, *FunctionDesc, *StreamHandlers) {
	fc, _ := ac.funcs[id]
	fd, _ := ac.fds[id]
	sths, _ := ac.sthss[id]
	return fc, fd, sths
}

func CurrentConnection(ctx context.Context) Connection {
	if ctx.Value("cn") == nil {
		return nil
	}
	cn, ok := ctx.Value("cn").(Connection)
	if !ok {
		return nil
	}
	return cn
}

func CurrentValues(ctx context.Context) map[string]any {
	if ctx.Value("values") == nil {
		return nil
	}
	values, ok := ctx.Value("values").(map[string]any)
	if !ok {
		return nil
	}
	return values
}

func CurrentFunction(ctx context.Context) stf.Function {
	if ctx.Value("values") == nil {
		return nil
	}
	values, ok := ctx.Value("values").(map[string]any)
	if !ok {
		return nil
	}
	fc, ok := values["fc"].(stf.Function)
	if !ok {
		return nil
	}
	return fc
}

func CurrentLogger(ctx context.Context) *slog.Logger {
	cn := CurrentConnection(ctx)
	if cn == nil {
		return nil
	}
	return cn.GetLogger()
}
