#include <MQTTPacket.h>
#include <boydemdb.h>

int main(int argc, char *argv[])
{
	MQTTPacket_connectData data = MQTTPacket_connectData_initializer;
	boydemdb db;

	boydemdb_set(db, "a", 2);
	MQTTDisconnect(NULL);

	return 0;
}
