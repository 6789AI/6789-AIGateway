import os
import sys

import paramiko


def main() -> int:
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=os.environ["CANVAS_DEPLOY_HOST"],
        port=int(os.environ["CANVAS_DEPLOY_PORT"]),
        username=os.environ["CANVAS_DEPLOY_USER"],
        password=os.environ["CANVAS_DEPLOY_SSH_PASSWORD"],
        timeout=15,
    )
    try:
        if sys.argv[1] == "exec":
            _, stdout, stderr = client.exec_command(sys.argv[2], timeout=3600)
            stdout.channel.set_combine_stderr(True)
            for chunk in iter(lambda: stdout.read(65536), b""):
                sys.stdout.buffer.write(chunk)
                sys.stdout.buffer.flush()
            return stdout.channel.recv_exit_status()
        if sys.argv[1] == "upload":
            with client.open_sftp() as sftp:
                sftp.put(sys.argv[2], sys.argv[3])
            return 0
        raise ValueError(f"unsupported operation: {sys.argv[1]}")
    finally:
        client.close()


if __name__ == "__main__":
    raise SystemExit(main())
