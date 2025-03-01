package job

import (
	"context"
	"fmt"

	comet "github.com/nanfengpo/goim/api/comet/grpc"
	pb "github.com/nanfengpo/goim/api/logic/grpc"
	"github.com/nanfengpo/goim/pkg/bytes"
	log "github.com/golang/glog"
)

func (j *Job) push(ctx context.Context, pushMsg *pb.PushMsg) (err error) {
	switch pushMsg.Type {
	case pb.PushMsg_PUSH:
		err = j.pushKeys(pushMsg.Operation, pushMsg.Server, pushMsg.Keys, pushMsg.Msg)
	case pb.PushMsg_ROOM:
		err = j.getRoom(pushMsg.Room).Push(pushMsg.Operation, pushMsg.Msg)
	case pb.PushMsg_BROADCAST:
		err = j.broadcast(pushMsg.Operation, pushMsg.Msg, pushMsg.Speed)
	default:
		err = fmt.Errorf("no match push type: %s", pushMsg.Type)
	}
	return
}

// pushKeys push a message to a batch of subkeys.
func (j *Job) pushKeys(operation int32, serverID string, subKeys []string, body []byte) (err error) {
	buf := bytes.NewWriterSize(len(body) + 64)
	p := &comet.Proto{
		Ver:  1,
		Op:   operation,
		Body: body,
	}
	p.WriteTo(buf)
	p.Body = buf.Buffer()
	p.Op = comet.OpRaw
	var args = comet.PushMsgReq{
		Keys:    subKeys,
		ProtoOp: operation,
		Proto:   p,
	}
	if c, ok := j.cometServers[serverID]; ok {
		if err = c.Push(&args); err != nil {
			log.Errorf("c.Push(%v) serverID:%s error(%v)", args, serverID, err)
		}
		log.Infof("pushKey:%s comets:%d", serverID, len(j.cometServers))
	}
	return
}

// broadcast broadcast a message to all.
func (j *Job) broadcast(operation int32, body []byte, speed int32) (err error) {
	buf := bytes.NewWriterSize(len(body) + 64)
	p := &comet.Proto{
		Ver:  1,
		Op:   operation,
		Body: body,
	}
	p.WriteTo(buf)
	p.Body = buf.Buffer()
	p.Op = comet.OpRaw
	comets := j.cometServers
	speed /= int32(len(comets))
	var args = comet.BroadcastReq{
		ProtoOp: operation,
		Proto:   p,
		Speed:   speed,
	}
	for serverID, c := range comets {
		if err = c.Broadcast(&args); err != nil {
			log.Errorf("c.Broadcast(%v) serverID:%s error(%v)", args, serverID, err)
		}
	}
	log.Infof("broadcast comets:%d", len(comets))
	return
}

// broadcastRoomRawBytes broadcast aggregation messages to room.
func (j *Job) broadcastRoomRawBytes(roomID string, body []byte) (err error) {
	args := comet.BroadcastRoomReq{
		RoomID: roomID,
		Proto: &comet.Proto{
			Ver:  1,
			Op:   comet.OpRaw,
			Body: body,
		},
	}
	comets := j.cometServers
	for serverID, c := range comets {
		if err = c.BroadcastRoom(&args); err != nil {
			log.Errorf("c.BroadcastRoom(%v) roomID:%s serverID:%s error(%v)", args, roomID, serverID, err)
		}
	}
	log.Infof("broadcastRoom comets:%d", len(comets))
	return
}
