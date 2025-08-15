package network 

import (
    "fmt"
    "net"
    "os"
    "os/exec"
    "strconv"
    "strings"
    
    "congo/internals/types"
)


func setupNetworking(config *Config) error {
    // Create bridge if it doesn't exist
    if err := createBridge(config.Network.Bridge); err != nil {
        return fmt.Errorf("failed to create bridge: %v", err)
    }

    // Create veth pair
    containerVeth := fmt.Sprintf("veth%d", os.Getpid())
    hostVeth := fmt.Sprintf("hveth%d", os.Getpid())

    if err := createVethPair(containerVeth, hostVeth); err != nil {
        return fmt.Errorf("failed to create veth pair: %v", err)
    }

    // Connect host veth to bridge
    if err := connectToBridge(hostVeth, config.Network.Bridge); err != nil {
        return fmt.Errorf("failed to connect to bridge: %v", err)
    }

    // Setup container network namespace
    if err := setupContainerNetNS(containerVeth, config.Network.ContainerIP); err != nil {
        return fmt.Errorf("failed to setup container network namespace: %v", err)
    }

    // Setup port forwarding
    if err := setupPortForwarding(config.Network.PortMaps, config.Network.ContainerIP); err != nil {
        return fmt.Errorf("failed to setup port forwarding: %v", err)
    }

    return nil
}

func createBridge(name string) error {
    // Check if bridge exists
    if _, err := net.InterfaceByName(name); err == nil {
        return nil // Bridge already exists
    }

    // Create bridge using ip command
    if err := exec.Command("ip", "link", "add", name, "type", "bridge").Run(); err != nil {
        return err
    }

    // Set bridge up
    if err := exec.Command("ip", "link", "set", name, "up").Run(); err != nil {
        return err
    }

    return nil
}

func createVethPair(container, host string) error {
    // Create veth pair
    if err := exec.Command("ip", "link", "add", container, "type", "veth", "peer", "name", host).Run(); err != nil {
        return err
    }

    // Set host interface up
    if err := exec.Command("ip", "link", "set", host, "up").Run(); err != nil {
        return err
    }

    return nil
}

func connectToBridge(veth, bridge string) error {
    return exec.Command("ip", "link", "set", veth, "master", bridge).Run()
}

func setupContainerNetNS(veth, ip string) error {
    // Move veth to container namespace
    if err := exec.Command("ip", "link", "set", veth, "netns", strconv.Itoa(os.Getpid())).Run(); err != nil {
        return err
    }

    // Setup loopback interface
    if err := exec.Command("ip", "link", "set", "lo", "up").Run(); err != nil {
        return err
    }

    // Setup container veth
    if err := exec.Command("ip", "link", "set", veth, "up").Run(); err != nil {
        return err
    }

    // Assign IP to container veth
    if err := exec.Command("ip", "addr", "add", ip, "dev", veth).Run(); err != nil {
        return err
    }

    return nil
}

func setupPortForwarding(portMaps []PortMapping, containerIP string) error {
    for _, port := range portMaps {
        // Add iptables DNAT rule for port forwarding
        rule := fmt.Sprintf(
            "-t nat -A PREROUTING -p %s --dport %d -j DNAT --to-destination %s:%d",
            port.Protocol,
            port.HostPort,
            containerIP,
            port.ContainerPort,
        )
        
        if err := exec.Command("iptables", strings.Split(rule, " ")...).Run(); err != nil {
            return fmt.Errorf("failed to add port forwarding rule: %v", err)
        }
    }
    return nil
}
